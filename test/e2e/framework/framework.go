package framework

import (
	"fmt"

	"github.com/amadeusitgroup/redis-operator/pkg/client/clientset/versioned"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Framework stores necessary info to run e2e
type Framework struct {
	KubeConfig *rest.Config
}

type frameworkContextType struct {
	KubeConfigPath string
	ImageTag       string
}

// FrameworkContext stores globally the framework context
var FrameworkContext frameworkContextType

// NewFramework creates and initializes the a Framework struct
func NewFramework() (*Framework, error) {
	Logf("KubeconfigPath-> %q", FrameworkContext.KubeConfigPath)
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", FrameworkContext.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve kubeConfig:%v", err)
	}
	return &Framework{
		KubeConfig: kubeConfig,
	}, nil
}

func (f *Framework) kubeClient() (clientset.Interface, error) {
	return clientset.NewForConfig(f.KubeConfig)
}

func (f *Framework) redisOperatorClient() (versioned.Interface, error) {
	c, err := versioned.NewForConfig(f.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create rediscluster client:%v", err)
	}
	return c, err
}
