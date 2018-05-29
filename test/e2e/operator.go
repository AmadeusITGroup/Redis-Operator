package e2e

import (
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	// for test lisibility
	. "github.com/onsi/ginkgo"
	// for test lisibility
	. "github.com/onsi/gomega"

	rapi "github.com/amadeusitgroup/redis-operator/pkg/api/redis/v1"
	"github.com/amadeusitgroup/redis-operator/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/redis-operator/test/e2e/framework"
)

func deleteRedisCluster(client versioned.Interface, rediscluster *rapi.RedisCluster) {
	client.Redisoperator().RedisClusters(rediscluster.Namespace).Delete(rediscluster.Name, &metav1.DeleteOptions{})
}

var redisClient versioned.Interface
var kubeClient clientset.Interface
var rediscluster *rapi.RedisCluster

const clusterName = "cluster1"
const clusterNs = api.NamespaceDefault

var _ = BeforeSuite(func() {
	redisClient, kubeClient = framework.BuildAndSetClients()
})

var _ = AfterSuite(func() {
	deleteRedisCluster(redisClient, rediscluster)
})

var _ = Describe("RedisCluster CRUD", func() {
	It("should create a RedisCluster", func() {

		rediscluster = framework.NewRedisCluster(clusterName, clusterNs, framework.FrameworkContext.ImageTag, 3, 1)

		Eventually(framework.HOCreateRedisNodeServiceAccount(kubeClient, rediscluster), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOCreateRedisCluster(redisClient, rediscluster, clusterNs), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsRedisClusterPodDisruptionBudgetCreated(kubeClient, rediscluster), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsRedisClusterStarted(redisClient, rediscluster, clusterNs), "5m", "5s").ShouldNot(HaveOccurred())

	})

	Context("when the Redis Cluster is created properly", func() {
		It("should scale up a RedisCluster", func() {
			nbMaster := int32(4)
			Eventually(framework.HOUpdateConfigRedisCluster(redisClient, rediscluster, &nbMaster, nil), "5s", "1s").ShouldNot(HaveOccurred())

			Eventually(framework.HOIsRedisClusterStarted(redisClient, rediscluster, clusterNs), "5m", "5s").ShouldNot(HaveOccurred())
		})
		Context("when the scale up successed", func() {
			It("should scale down a RedisCluster", func() {
				nbMaster := int32(3)
				Eventually(framework.HOUpdateConfigRedisCluster(redisClient, rediscluster, &nbMaster, nil), "5s", "1s").ShouldNot(HaveOccurred())

				Eventually(framework.HOIsRedisClusterStarted(redisClient, rediscluster, clusterNs), "5m", "5s").ShouldNot(HaveOccurred())
			})
			Context("when the scale down successed", func() {
				It("should increase the a RedisCluster's replica factor ", func() {
					replicas := int32(2)
					Eventually(framework.HOUpdateConfigRedisCluster(redisClient, rediscluster, nil, &replicas), "5s", "1s").ShouldNot(HaveOccurred())

					Eventually(framework.HOIsRedisClusterStarted(redisClient, rediscluster, clusterNs), "5m", "5s").ShouldNot(HaveOccurred())
				})

				It("should decrease the a RedisCluster's replica factor ", func() {
					replicas := int32(1)
					Eventually(framework.HOUpdateConfigRedisCluster(redisClient, rediscluster, nil, &replicas), "5s", "1s").ShouldNot(HaveOccurred())

					Eventually(framework.HOIsRedisClusterStarted(redisClient, rediscluster, clusterNs), "5m", "5s").ShouldNot(HaveOccurred())
				})
			})
		})
		It("should update the RedisCluster", func() {
			newTag := "4.0"
			rediscluster = framework.NewRedisCluster(clusterName, clusterNs, newTag, 3, 1)

			Eventually(framework.HOUpdateRedisCluster(redisClient, rediscluster, clusterNs), "5s", "1s").ShouldNot(HaveOccurred())

			Eventually(framework.HOIsPodSpecUpdated(kubeClient, rediscluster, newTag), "7m", "5s").ShouldNot(HaveOccurred())

			Eventually(framework.HOIsRedisClusterStarted(redisClient, rediscluster, clusterNs), "7m", "5s").ShouldNot(HaveOccurred())
		})
	})

})
