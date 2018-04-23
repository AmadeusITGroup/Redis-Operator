package redisnode

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/golang/glog"

	"github.com/heptiolabs/healthcheck"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	radix "github.com/mediocregopher/radix.v2/redis"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/amadeusitgroup/redis-operator/pkg/redis"
	"github.com/amadeusitgroup/redis-operator/pkg/utils"
)

// RedisNode constains all info to run the redis-node.
type RedisNode struct {
	config     *Config
	kubeClient clientset.Interface
	redisAdmin redis.AdminInterface
	admOptions redis.AdminOptions

	// Kubernetes Probes handler
	health healthcheck.Handler

	httpServer *http.Server
}

// NewRedisNode builds and returns new RedisNode instance
func NewRedisNode(cfg *Config) *RedisNode {
	kubeConfig, err := initKubeConfig(cfg)
	if err != nil {
		glog.Fatalf("Unable to init rediscluster controller: %v", err)
	}

	kubeClient, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to initialize kubeClient:%v", err)
	}

	rn := &RedisNode{
		config:     cfg,
		kubeClient: kubeClient,
	}

	return rn
}

// Run executes the RedisNode
func (r *RedisNode) Run(stop <-chan struct{}) error {
	node, err := r.init()
	if err != nil {
		return err
	}

	go r.runHTTPServer(stop)

	node, err = r.run(node)
	if err != nil {
		return err
	}

	glog.Info("Awaiting stop signal")
	<-stop
	glog.Info("Receive Stop Signal...")

	return r.handleStop(node)
}

func initKubeConfig(c *Config) (*rest.Config, error) {
	if len(c.KubeConfigFile) > 0 {
		return clientcmd.BuildConfigFromFlags(c.Master, c.KubeConfigFile) // out of cluster config
	}
	return rest.InClusterConfig()
}

func (r *RedisNode) init() (*Node, error) {
	// Too fast restart of redis-server can result in slots lost
	// This is due to a possible bug in Redis. Redis don't check that the node id behind an IP is still the same after a deconnection/reconnection.
	// And so the the slave reconnect and sync to an empty node in our case
	// Therefore, we need to wait for the possible failover to finish.
	// 2*nodetimeout for failed state detection, 2*nodetimeout for voting, 2*nodetimeout for safety
	time.Sleep(r.config.RedisStartDelay)

	nodesAddr, err := getRedisNodesAddrs(r.kubeClient, r.config.Cluster.Namespace, r.config.Cluster.NodeService)
	if err != nil {
		glog.Warning(err)
	}

	if len(nodesAddr) > 0 {
		if glog.V(3) {
			for _, node := range nodesAddr {
				glog.Info("REDIS Node addresses:", node)
			}
		}
	} else {
		glog.Info("Redis Node list empty")
	}

	r.admOptions = redis.AdminOptions{
		ConnectionTimeout:  time.Duration(r.config.Redis.DialTimeout) * time.Millisecond,
		RenameCommandsFile: r.config.Redis.GetRenameCommandsFile(),
	}
	host, err := os.Hostname()
	if err != nil {
		r.admOptions.ClientName = host // will be pod name in kubernetes
	}

	r.redisAdmin = redis.NewAdmin(nodesAddr, &r.admOptions)

	me := NewNode(r.config, r.redisAdmin)
	if me == nil {
		glog.Fatal("Unable to get Node information")
	}
	defer me.Clear()

	// reconfigure redis config file with proper IP/port
	err = me.UpdateNodeConfigFile()
	if err != nil {
		glog.Fatal("Unable to update the configuration file, err:", err)
	}

	me.ClearDataFolder() // may be needed if container crashes and restart at the same place

	r.httpServer = &http.Server{Addr: r.config.HTTPServerAddr}
	if err := r.configureHealth(); err != nil {
		glog.Errorf("unable to configure health checks, err:%v", err)
		return nil, err
	}

	return me, nil
}

func (r *RedisNode) run(me *Node) (*Node, error) {
	// Start redis server and wait for it to be accessible
	chRedis := make(chan error)
	go WrapRedis(r.config, chRedis)
	starterr := testAndWaitConnection(me.Addr, r.config.RedisStartWait)
	if starterr != nil {
		glog.Error("Error while waiting for redis to start: ", starterr)
		return nil, starterr
	}

	configFunc := func() (bool, error) {
		// Initial redis server configuration
		nodes, initCluster := r.isClusterInitialization(me.Addr)

		if initCluster {
			glog.Infof("Initializing cluster with slots from 0 to %d", redis.HashMaxSlots)
			if err := me.InitRedisCluster(me.Addr); err != nil {
				glog.Error("Unable to init the cluster with this node, err:", err)
				return false, err
			}
		} else {
			glog.Infof("Attaching node to cluster")
			r.redisAdmin.RebuildConnectionMap(nodes, &r.admOptions)
			if err := me.AttachNodeToCluster(me.Addr); err != nil {
				glog.Error("Unable to attach a node to the cluster, err:", err)
				return false, nil
			}
		}
		return true, nil
	}
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFunc()
	wait.PollUntil(2*time.Second, configFunc, ctx.Done())

	glog.Infof("RedisNode: Runnning properly")
	return me, nil
}

func (r *RedisNode) isClusterInitialization(currentIP string) ([]string, bool) {
	var initCluster = true
	nodesAddr, _ := getRedisNodesAddrs(r.kubeClient, r.config.Cluster.Namespace, r.config.Cluster.NodeService)
	if len(nodesAddr) > 0 {
		initCluster = false
		if glog.V(3) {
			for _, node := range nodesAddr {
				glog.Info("REDIS Node addresses:", node)
			}
		}
	} else {
		glog.Info("Redis Node list empty")
	}

	if len(nodesAddr) == 1 && nodesAddr[0] == net.JoinHostPort(currentIP, r.config.RedisServerPort) {
		// Init Master cluster
		initCluster = true
	}

	return nodesAddr, initCluster
}

func (r *RedisNode) handleStop(me *Node) error {
	nodesAddr, err := getRedisNodesAddrs(r.kubeClient, r.config.Cluster.Namespace, r.config.Cluster.NodeService)
	if err != nil {
		glog.Error("Unable to retrieve Redis Node, err:", err)
		return err
	}

	r.redisAdmin.Connections().ReplaceAll(nodesAddr)
	if err = me.StartFailover(); err != nil {
		glog.Errorf("Failover node:%s  error:%s", me.Addr, err)
	}

	if err = me.ForgetNode(); err != nil {
		glog.Errorf("Forget node:%s  error:%s", me.Addr, err)
	}

	return err
}

func (r *RedisNode) configureHealth() error {
	ip, err := utils.GetMyIP()
	if err != nil {
		glog.Errorf("unable to get my IP, err:%v", err)
		return err
	}
	addr := net.JoinHostPort(ip, r.config.RedisServerPort)

	health := healthcheck.NewHandler()
	health.AddReadinessCheck("Check redis-node readiness", func() error {
		err = readinessCheck(addr)
		if err != nil {
			glog.Errorf("readiness check failed, err:%v", err)
			return err
		}
		return nil
	})

	health.AddLivenessCheck("Check redis-node liveness", func() error {
		err = livenessCheck(addr)
		if err != nil {
			glog.Errorf("liveness check failed, err:%v", err)
			return err
		}
		return nil
	})

	r.health = health
	http.Handle("/", r.health)
	http.Handle("/metrics", promhttp.Handler())
	return nil
}

func readinessCheck(addr string) error {
	client, rediserr := redis.NewClient(addr, time.Second, map[string]string{}) // will fail if node not accessible or slot range not set
	if rediserr != nil {
		return fmt.Errorf("Readiness failed, err: %v", rediserr)
	}
	defer client.Close()
	array, err := client.Cmd("CLUSTER", "SLOTS").Array()
	if err != nil {
		return fmt.Errorf("Readiness failed, cluster slots response err: %v", rediserr)
	}

	if len(array) == 0 {
		return fmt.Errorf("Readiness failed, cluster slots response empty")
	}
	glog.V(6).Info("Readiness probe ok")
	return nil
}

func livenessCheck(addr string) error {
	client, rediserr := redis.NewClient(addr, time.Second, map[string]string{}) // will fail if node not accessible or slot range not set
	if rediserr != nil {
		return fmt.Errorf("Liveness failed, err: %v", rediserr)
	}
	defer client.Close()
	glog.V(6).Info("Liveness probe ok")
	return nil
}

func (r *RedisNode) runHTTPServer(stop <-chan struct{}) error {

	go func() {
		glog.Info("Listening on http://%s\n", r.httpServer.Addr)

		if err := r.httpServer.ListenAndServe(); err != nil {
			glog.Error("Http server error: ", err)
		}
	}()

	<-stop
	glog.Info("Shutting down the http server...")
	return r.httpServer.Shutdown(context.Background())
}

// WrapRedis start a redis server in a sub process
func WrapRedis(c *Config, ch chan error) {
	cmd := exec.Command(c.RedisServerBin, c.Redis.ConfigFile)
	cmd.Stdout = utils.NewLogWriter(glog.Info)
	cmd.Stderr = utils.NewLogWriter(glog.Error)

	if err := cmd.Start(); err != nil {
		glog.Error("Error during redis-server start, err", err)
		ch <- err
	}

	if err := cmd.Wait(); err != nil {
		glog.Error("Error during redis-server execution, err", err)
		ch <- err
	}

	glog.Info("Redis-server stop properly")
	ch <- nil
}

func testAndWaitConnection(addr string, maxWait time.Duration) error {
	startTime := time.Now()
	waitTime := maxWait
	for {
		currentTime := time.Now()
		timeout := waitTime - startTime.Sub(currentTime)
		if timeout <= 0 {
			return errors.New("Timeout reached")
		}
		client, err := radix.DialTimeout("tcp", addr, timeout)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		defer client.Close()

		if resp, err := client.Cmd("PING").Str(); err != nil {
			client.Close()
			time.Sleep(100 * time.Millisecond)
			continue
		} else if resp != "PONG" {
			client.Close()
			time.Sleep(100 * time.Millisecond)
			continue
		}

		return nil
	}
}

func getRedisNodesAddrs(kubeClient clientset.Interface, namespace, service string) ([]string, error) {
	addrs := []string{}
	eps, err := kubeClient.Core().Endpoints(namespace).Get(service, meta_v1.GetOptions{})
	if err != nil {
		return addrs, err
	}

	for _, subset := range eps.Subsets {
		for _, host := range subset.Addresses {
			for _, port := range subset.Ports {
				addrs = append(addrs, net.JoinHostPort(host.IP, strconv.Itoa(int(port.Port))))
			}
		}
	}

	return addrs, nil
}
