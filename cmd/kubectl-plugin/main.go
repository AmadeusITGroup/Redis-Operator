package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
	"github.com/olekukonko/tablewriter"

	kapiv1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	v1 "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	rclient "github.com/amadeusitgroup/redis-operator/pkg/client"
)

func main() {
	cmdBin := "kubectl"
	if val := os.Getenv("KUBECTL_PLUGINS_CALLER"); val != "" {
		cmdBin = val
	}

	namespace := "default"
	if val := os.Getenv("KUBECTL_PLUGINS_CURRENT_NAMESPACE"); val != "" {
		namespace = val
	}

	clusterName := ""
	if val := os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_RC"); val != "" {
		clusterName = val
	}

	kubeConfigBytes, err := exec.Command(cmdBin, "config", "view").Output()
	if err != nil {
		log.Fatal(err)
	}

	tmpConf, err := ioutil.TempFile("", "example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(tmpConf.Name()) // clean up
	if _, err = tmpConf.Write(kubeConfigBytes); err != nil {
		log.Fatal(err)
	}
	// use the current context in kubeconfig
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", tmpConf.Name())
	if err != nil {
		panic(err.Error())
	}

	redisClient, err := rclient.NewClient(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to init redis.clientset from kubeconfig:%v", err)
	}

	var rcs *v1.RedisClusterList
	if clusterName == "" {
		rcs, err = redisClient.RedisV1().RedisClusters(namespace).List(meta_v1.ListOptions{})
		if err != nil {
			glog.Fatalf("unable to list rediscluster err:%v", err)
		}
	} else {
		rcs = &v1.RedisClusterList{}
		rc, err := redisClient.RedisV1().RedisClusters(namespace).Get(clusterName, meta_v1.GetOptions{})
		if err != nil {
			glog.Fatalf("unable to list rediscluster err:%v", err)
		}
		rcs.Items = append(rcs.Items, *rc)
	}

	data := [][]string{}
	for _, rc := range rcs.Items {
		data = append(data, []string{rc.Name, rc.Namespace, buildPodStatus(&rc), buildClusterStatus(&rc), string(rc.Status.Cluster.Status), buildMasterStatus(&rc), buildReplicationStatus(&rc)})
	}

	if len(data) == 0 {
		fmt.Println("No resources found.")
		os.Exit(0)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "Pods", "Ops Status", "Redis Status", "Nb Master", "Replication"})
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)

	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output

	os.Exit(0)
}

func hasStatus(rc *v1.RedisCluster, conditionType v1.RedisClusterConditionType, status kapiv1.ConditionStatus) bool {
	for _, cond := range rc.Status.Conditions {
		if cond.Type == conditionType && cond.Status == status {
			return true
		}
	}
	return false
}

func buildClusterStatus(rc *v1.RedisCluster) string {
	status := []string{}

	if hasStatus(rc, v1.RedisClusterOK, kapiv1.ConditionFalse) {
		status = append(status, "KO")
	} else if hasStatus(rc, v1.RedisClusterOK, kapiv1.ConditionTrue) {
		status = append(status, string(v1.RedisClusterOK))
	}

	if hasStatus(rc, v1.RedisClusterRollingUpdate, kapiv1.ConditionTrue) {
		status = append(status, string(v1.RedisClusterRollingUpdate))
	} else if hasStatus(rc, v1.RedisClusterScaling, kapiv1.ConditionTrue) {
		status = append(status, string(v1.RedisClusterScaling))
	} else if hasStatus(rc, v1.RedisClusterRebalancing, kapiv1.ConditionTrue) {
		status = append(status, string(v1.RedisClusterRebalancing))
	}

	return strings.Join(status, "-")
}

func buildPodStatus(rc *v1.RedisCluster) string {
	specMaster := *rc.Spec.NumberOfMaster
	specReplication := *rc.Spec.ReplicationFactor
	podWanted := (1 + specReplication) * specMaster

	pod := rc.Status.Cluster.NbPods
	podReady := rc.Status.Cluster.NbPodsReady

	return fmt.Sprintf("%d/%d/%d", podReady, pod, podWanted)
}

func buildMasterStatus(rc *v1.RedisCluster) string {
	return fmt.Sprintf("%d/%d", rc.Status.Cluster.NumberOfMaster, *rc.Spec.NumberOfMaster)
}

func buildReplicationStatus(rc *v1.RedisCluster) string {
	spec := *rc.Spec.ReplicationFactor
	return fmt.Sprintf("%d-%d/%d", rc.Status.Cluster.MinReplicationFactor, rc.Status.Cluster.MaxReplicationFactor, spec)
}
