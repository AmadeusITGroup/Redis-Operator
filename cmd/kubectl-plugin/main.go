package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang/glog"
	"github.com/olekukonko/tablewriter"

	kapiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	v1 "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	rclient "github.com/amadeusitgroup/redis-operator/pkg/client"
)

func main() {
	namespace := ""
	if val := os.Getenv("KUBECTL_PLUGINS_CURRENT_NAMESPACE"); val != "" {
		namespace = val
	}

	clusterName := ""
	if val := os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_RC"); val != "" {
		clusterName = val
	}

	kubeconfigFilePath := getKubeConfigDefaultPath(getHomePath())
	if len(kubeconfigFilePath) == 0 {
		log.Fatal("error initializing config. The KUBECONFIG environment variable must be defined.")
	}

	config, err := configFromPath(kubeconfigFilePath)
	if err != nil {
		log.Fatalf("error obtaining kubectl config: %v", err)
	}

	rest, err := config.ClientConfig()
	if err != nil {
		log.Fatalf(err.Error())
	}

	redisClient, err := rclient.NewClient(rest)
	if err != nil {
		glog.Fatalf("Unable to init redis.clientset from kubeconfig:%v", err)
	}

	rcs := &v1.RedisClusterList{}
	if clusterName == "" {
		rcs, err = redisClient.RedisoperatorV1().RedisClusters(namespace).List(meta_v1.ListOptions{})
		if err != nil {
			glog.Fatalf("unable to list redisclusters:%v", err)
		}
	} else {
		rc, err := redisClient.RedisoperatorV1().RedisClusters(namespace).Get(clusterName, meta_v1.GetOptions{})
		if err == nil && rc != nil {
			rcs.Items = append(rcs.Items, *rc)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			glog.Fatalf("unable to get rediscluster %s: %v", clusterName, err)
		}
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

func configFromPath(path string) (clientcmd.ClientConfig, error) {
	rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: path}
	credentials, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("the provided credentials %q could not be loaded: %v", path, err)
	}

	overrides := &clientcmd.ConfigOverrides{
		Context: clientcmdapi.Context{
			Namespace: os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_NAMESPACE"),
		},
	}

	context := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONTEXT")
	if len(context) > 0 {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		return clientcmd.NewNonInteractiveClientConfig(*credentials, context, overrides, rules), nil
	}
	return clientcmd.NewDefaultClientConfig(*credentials, overrides), nil
}

func getHomePath() string {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
	}

	return home
}

func getKubeConfigDefaultPath(home string) string {
	kubeconfig := filepath.Join(home, ".kube", "config")

	kubeconfigEnv := os.Getenv("KUBECONFIG")
	if len(kubeconfigEnv) > 0 {
		kubeconfig = kubeconfigEnv
	}

	configFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONFIG")
	kubeConfigFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_KUBECONFIG")
	if len(configFile) > 0 {
		kubeconfig = configFile
	} else if len(kubeConfigFile) > 0 {
		kubeconfig = kubeConfigFile
	}

	return kubeconfig
}
