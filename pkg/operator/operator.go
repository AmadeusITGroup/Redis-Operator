package operator

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/heptiolabs/healthcheck"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/amadeusitgroup/redis-operator/pkg/client"
	versionedclient "github.com/amadeusitgroup/redis-operator/pkg/client/clientset/versioned"
	redisinformers "github.com/amadeusitgroup/redis-operator/pkg/client/informers/externalversions"
	"github.com/amadeusitgroup/redis-operator/pkg/controller"
	"github.com/amadeusitgroup/redis-operator/pkg/garbagecollector"
)

// RedisOperator constains all info to run the redis operator.
type RedisOperator struct {
	kubeInformerFactory  kubeinformers.SharedInformerFactory
	redisInformerFactory redisinformers.SharedInformerFactory

	controller *controller.Controller
	GC         garbagecollector.Interface

	// Kubernetes Probes handler
	health healthcheck.Handler

	httpServer *http.Server
}

// NewRedisOperator builds and returns new RedisOperator instance
func NewRedisOperator(cfg *Config) *RedisOperator {
	kubeConfig, err := initKubeConfig(cfg)
	if err != nil {
		glog.Fatalf("Unable to init rediscluster controller: %v", err)
	}

	// apiextensionsclientset, err :=
	extClient, err := apiextensionsclient.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to init clientset from kubeconfig:%v", err)
	}

	_, err = client.DefineRedisClusterResource(extClient)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		glog.Fatalf("Unable to define RedisCluster resource:%v", err)
	}

	kubeClient, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to initialize kubeClient:%v", err)
	}

	redisClient, err := versionedclient.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to init redis.clientset from kubeconfig:%v", err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	redisInformerFactory := redisinformers.NewSharedInformerFactory(redisClient, time.Second*30)

	op := &RedisOperator{
		kubeInformerFactory:  kubeInformerFactory,
		redisInformerFactory: redisInformerFactory,
		controller:           controller.NewController(controller.NewConfig(1, cfg.Redis), kubeClient, redisClient, kubeInformerFactory, redisInformerFactory),
		GC:                   garbagecollector.NewGarbageCollector(redisClient, kubeClient, redisInformerFactory),
	}

	op.configureHealth()
	op.httpServer = &http.Server{Addr: cfg.ListenAddr, Handler: op.health}

	return op
}

// Run executes the Redis Operator
func (op *RedisOperator) Run(stop <-chan struct{}) error {
	var err error
	if op.controller != nil {
		op.kubeInformerFactory.Start(stop)
		op.redisInformerFactory.Start(stop)
		go op.runHTTPServer(stop)
		op.runGC(stop)
		err = op.controller.Run(stop)
	}

	return err
}

func (op *RedisOperator) runGC(stop <-chan struct{}) {
	go func() {
		if !cache.WaitForCacheSync(stop, op.GC.InformerSync()) {
			glog.Errorf("Timed out waiting for caches to sync")
		}
		wait.Until(func() {
			err := op.GC.CollectRedisClusterGarbage()
			if err != nil {
				glog.Errorf("collecting rediscluster pods and services: %v", err)
			}
		}, garbagecollector.Interval, stop)
		// run garbage collection before stopping the process
		op.GC.CollectRedisClusterGarbage()
	}()
}

func initKubeConfig(c *Config) (*rest.Config, error) {
	if len(c.KubeConfigFile) > 0 {
		return clientcmd.BuildConfigFromFlags(c.Master, c.KubeConfigFile) // out of cluster config
	}
	return rest.InClusterConfig()
}

func (op *RedisOperator) configureHealth() {
	op.health = healthcheck.NewHandler()
	op.health.AddReadinessCheck("RedisCluster_cache_sync", func() error {
		if op.controller.RedisClusterSynced() {
			return nil
		}
		return fmt.Errorf("RedisCluster cache not sync")
	})
	op.health.AddReadinessCheck("Pod_cache_sync", func() error {
		if op.controller.PodSynced() {
			return nil
		}
		return fmt.Errorf("Pod cache not sync")
	})
	op.health.AddReadinessCheck("Service_cache_sync", func() error {
		if op.controller.ServiceSynced() {
			return nil
		}
		return fmt.Errorf("Service cache not sync")
	})
	op.health.AddReadinessCheck("PodDiscruptionBudget_cache_sync", func() error {
		if op.controller.PodDiscruptionBudgetSynced() {
			return nil
		}
		return fmt.Errorf("PodDiscruptionBudget cache not sync")
	})
}

func (op *RedisOperator) runHTTPServer(stop <-chan struct{}) error {

	go func() {
		glog.Info("Listening on http://%s\n", op.httpServer.Addr)

		if err := op.httpServer.ListenAndServe(); err != nil {
			glog.Error("Http server error: ", err)
		}
	}()

	<-stop
	glog.Info("Shutting down the http server...")
	return op.httpServer.Shutdown(context.Background())
}
