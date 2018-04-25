package framework

import (
	"fmt"
	"strings"

	"github.com/golang/glog"

	. "github.com/onsi/gomega"

	"github.com/amadeusitgroup/redis-operator/pkg/api/redis"
	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/client/clientset/versioned"

	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"
)

// NewRedisCluster builds and returns a new RedisCluster instance
func NewRedisCluster(name, namespace, tag string, nbMaster, replication int32) *rapi.RedisCluster {
	return &rapi.RedisCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       rapi.ResourceKind,
			APIVersion: redisk8soperatorio.GroupName + "/" + rapi.ResourceVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: rapi.RedisClusterSpec{
			AdditionalLabels:  map[string]string{"foo": "bar"},
			NumberOfMaster:    &nbMaster,
			ReplicationFactor: &replication,
			PodTemplate: &v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "redis-cluster",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "redis-node",
					Volumes: []v1.Volume{
						{Name: "data"},
					},
					Containers: []v1.Container{
						{
							Name:            "redis",
							Image:           fmt.Sprintf("redisoperator/redisnode:%s", tag),
							ImagePullPolicy: v1.PullIfNotPresent,
							Args: []string{
								"--v=6",
								"--logtostderr=true",
								"--alsologtostderr=true",
								fmt.Sprintf("--rs=%s", name),
								"--t=10s",
								"--d=10s",
								"--ns=$(POD_NAMESPACE)",
								"--cluster-node-timeout=2000",
							},
							Ports: []v1.ContainerPort{
								{Name: "redis", ContainerPort: 6379},
								{Name: "cluster", ContainerPort: 16379},
							},
							VolumeMounts: []v1.VolumeMount{
								{Name: "data", MountPath: "/redis-data"},
							},
							Env: []v1.EnvVar{
								{Name: "POD_NAMESPACE", ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
							},
							LivenessProbe: &v1.Probe{
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path: "/live",
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 12,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    30,
							},
							ReadinessProbe: &v1.Probe{
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path: "/ready",
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 12,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
						},
					},
				},
			},
		},
	}
}

// BuildAndSetClients builds and initilize rediscluster and kube client
func BuildAndSetClients() (versioned.Interface, clientset.Interface) {
	f, err := NewFramework()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(f).ShouldNot(BeNil())

	kubeClient, err := f.kubeClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(kubeClient).ShouldNot(BeNil())
	Logf("Check whether RedisCluster resource is registered...")

	redisClient, err := f.redisOperatorClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(redisClient).ShouldNot(BeNil())
	return redisClient, kubeClient
}

// HOCreateRedisCluster is an higher order func that returns the func to create a RedisCluster
func HOCreateRedisCluster(client versioned.Interface, rediscluster *rapi.RedisCluster, namespace string) func() error {
	return func() error {
		if _, err := client.RedisoperatorV1().RedisClusters(namespace).Create(rediscluster); err != nil {
			glog.Warningf("cannot create RedisCluster %s/%s: %v", namespace, rediscluster.Name, err)
			return err
		}
		Logf("RedisCluster created")
		return nil
	}
}

// HOUpdateRedisCluster is an higher order func that returns the func to update a RedisCluster
func HOUpdateRedisCluster(client versioned.Interface, rediscluster *rapi.RedisCluster, namespace string) func() error {
	return func() error {
		cluster, err := client.RedisoperatorV1().RedisClusters(rediscluster.Namespace).Get(rediscluster.Name, metav1.GetOptions{})
		if err != nil {
			Logf("Cannot get rediscluster:%v", err)
			return err
		}
		cluster.Spec = rediscluster.Spec
		if _, err := client.RedisoperatorV1().RedisClusters(namespace).Update(cluster); err != nil {
			glog.Warningf("cannot update RedisCluster %s/%s: %v", namespace, rediscluster.Name, err)
			return err
		}
		Logf("RedisCluster updated")
		return nil
	}
}

// HOIsRedisClusterStarted is an higher order func that returns the func that checks whether RedisCluster is started and configured properly
func HOIsRedisClusterStarted(client versioned.Interface, rediscluster *rapi.RedisCluster, namespace string) func() error {
	return func() error {
		cluster, err := client.RedisoperatorV1().RedisClusters(rediscluster.Namespace).Get(rediscluster.Name, metav1.GetOptions{})
		if err != nil {
			Logf("Cannot get rediscluster:%v", err)
			return err
		}

		if cluster.Status.Cluster.NumberOfMaster != *cluster.Spec.NumberOfMaster {
			return LogAndReturnErrorf("RedisCluster %s wrong configuration number of master spec: %d, current:%d", cluster.Name, *cluster.Spec.NumberOfMaster, cluster.Status.Cluster.NumberOfMaster)
		}

		if cluster.Spec.ReplicationFactor == nil {
			return LogAndReturnErrorf("RedisCluster %s is spec not updated", cluster.Name)
		}

		if cluster.Status.Cluster.MinReplicationFactor != *cluster.Spec.ReplicationFactor {
			return LogAndReturnErrorf("RedisCluster %s wrong configuration  min replication factor: %v, current: %v ", cluster.Name, *cluster.Spec.ReplicationFactor, rediscluster.Status.Cluster.MinReplicationFactor)
		}

		if cluster.Status.Cluster.MaxReplicationFactor != *cluster.Spec.ReplicationFactor {
			return LogAndReturnErrorf("RedisCluster %s wrong configuration  number of master spec: %d, wanted: %v ", cluster.Name, *cluster.Spec.ReplicationFactor, rediscluster.Status.Cluster.MaxReplicationFactor)
		}

		if cluster.Status.Cluster.Status != rapi.ClusterStatusOK {
			return LogAndReturnErrorf("RedisCluster %s status not OK, current value:%s", cluster.Status.Cluster.Status)
		}

		return nil
	}
}

// HOUpdateConfigRedisCluster is an higher order func that returns the func to update the RedisCluster configuration
func HOUpdateConfigRedisCluster(client versioned.Interface, rediscluster *rapi.RedisCluster, nbmaster, replicas *int32) func() error {
	return func() error {
		cluster, err := client.RedisoperatorV1().RedisClusters(rediscluster.Namespace).Get(rediscluster.Name, metav1.GetOptions{})
		if err != nil {
			glog.Warningf("cannot get RedisCluster %s/%s: %v", rediscluster.Namespace, rediscluster.Name, err)
			return err
		}
		if nbmaster != nil {
			rediscluster.Spec.NumberOfMaster = nbmaster
			cluster.Spec.NumberOfMaster = nbmaster
		}
		if replicas != nil {
			rediscluster.Spec.ReplicationFactor = replicas
			cluster.Spec.ReplicationFactor = replicas
		}
		if _, err := client.RedisoperatorV1().RedisClusters(rediscluster.Namespace).Update(cluster); err != nil {
			Logf("cannot update RedisCluster %s/%s: %v", rediscluster.Namespace, rediscluster.Name, err)
			return err
		}

		Logf("RedisCluster created")
		return nil
	}
}

// HOIsPodSpecUpdated use to check if a all the RedisCluster pod have the new PodSpec
func HOIsPodSpecUpdated(client clientset.Interface, rediscluster *rapi.RedisCluster, imageTag string) func() error {
	return func() error {
		labelSet := labels.Set{}
		labelSet[rapi.ClusterNameLabelKey] = rediscluster.Name
		podList, err := client.Core().Pods(rediscluster.Namespace).List(metav1.ListOptions{LabelSelector: labelSet.AsSelector().String()})
		if err != nil {
			Logf("cannot get RedisCluster %s/%s: %v", rediscluster.Namespace, rediscluster.Name, err)
			return err
		}

		for _, pod := range podList.Items {
			found := false
			for _, container := range pod.Spec.Containers {
				if container.Name == "redis" {
					found = true
					splitString := strings.Split(container.Image, ":")
					if len(splitString) != 2 {
						return fmt.Errorf("unable to get the tag from the container.Image:%s", container.Image)
					}
					if splitString[1] != imageTag {
						return fmt.Errorf("current container.Image have a wrong tag:%s, want:%s", splitString[1], imageTag)
					}
				}
			}
			if !found {
				return fmt.Errorf("unable to found the container with name: redis")
			}
		}

		Logf("RedisCluster podSpec updated properly")
		return nil
	}
}

// HOCreateRedisNodeServiceAccount  is an higher order func that returns the func to create the serviceaccount assiated to the redis-node pod.
func HOCreateRedisNodeServiceAccount(client clientset.Interface, rediscluster *rapi.RedisCluster) func() error {
	return func() error {
		_, err := client.Core().ServiceAccounts(rediscluster.Namespace).Get("redis-node", metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			newSa := v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: "redis-node",
				},
			}
			_, err = client.Core().ServiceAccounts(rediscluster.Namespace).Create(&newSa)
			if err != nil {
				return err
			}
		}

		_, err = client.Rbac().ClusterRoles().Get("redis-node", metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			cr := rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "redis-node",
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces", "services", "endpoints", "pods"},
						Verbs:     []string{"list", "get"},
					},
				},
			}
			_, err = client.Rbac().ClusterRoles().Create(&cr)
			if err != nil {
				return err
			}
		}
		_, err = client.Rbac().RoleBindings(rediscluster.Namespace).Get("redis-node", metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			rb := rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "redis-node",
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "redis-node",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "redis-node",
						Namespace: rediscluster.Namespace,
					},
				},
			}
			_, err = client.Rbac().RoleBindings(rediscluster.Namespace).Create(&rb)
			if err != nil {
				return err
			}
		}
		return nil
	}
}
