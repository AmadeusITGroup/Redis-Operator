package sanitycheck

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/errors"

	"github.com/amadeusitgroup/redis-operator/pkg/config"
	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

// FixClusterSplit use to detect and fix Cluster split
func FixClusterSplit(admin redis.AdminInterface, config *config.Redis, infos *redis.ClusterInfos, dryRun bool) (bool, error) {
	clusters := buildClustersLists(infos)

	if len(clusters) > 1 {
		if dryRun {
			return true, nil
		}
		return true, reassignClusters(admin, config, clusters)
	}
	glog.V(3).Info("[SanityChecks] No split cluster detected")
	return false, nil
}

type cluster []string

func reassignClusters(admin redis.AdminInterface, config *config.Redis, clusters []cluster) error {
	glog.Error("[SanityChecks] Cluster split detected, the Redis manager will recover from the issue, but data may be lost")
	var errs []error
	// only one cluster may remain
	mainCluster, badClusters := splitMainCluster(clusters)
	if len(mainCluster) == 0 {
		glog.Error("[SanityChecks] Impossible to fix cluster split, cannot elect main cluster")
		return fmt.Errorf("Impossible to fix cluster split, cannot elect main cluster")
	}
	glog.Infof("[SanityChecks] Cluster '%s' is elected as main cluster", mainCluster)
	// reset admin to connect to the correct cluster
	admin.Connections().ReplaceAll(mainCluster)

	// reconfigure bad clusters
	for _, cluster := range badClusters {
		glog.Warningf("[SanityChecks] All keys stored in redis cluster '%s' will be lost", cluster)
		clusterAdmin := redis.NewAdmin(cluster,
			&redis.AdminOptions{
				ConnectionTimeout:  time.Duration(config.DialTimeout) * time.Millisecond,
				RenameCommandsFile: config.GetRenameCommandsFile(),
			})
		for _, nodeAddr := range cluster {
			if err := clusterAdmin.FlushAndReset(nodeAddr, redis.ResetHard); err != nil {
				glog.Errorf("unable to flush the node: %s, err:%v", nodeAddr, err)
				errs = append(errs, err)
			}
			if err := admin.AttachNodeToCluster(nodeAddr); err != nil {
				glog.Errorf("unable to attach the node: %s, err:%v", nodeAddr, err)
				errs = append(errs, err)
			}

		}
		clusterAdmin.Close()
	}

	return errors.NewAggregate(errs)
}

func splitMainCluster(clusters []cluster) (cluster, []cluster) {
	if len(clusters) == 0 {
		return cluster{}, []cluster{}
	}
	// only the bigger cluster is kept, or the first one if several cluster have the same size
	maincluster := -1
	maxSize := 0
	for i, c := range clusters {
		if len(c) > maxSize {
			maxSize = len(c)
			maincluster = i
		}
	}
	if maincluster != -1 {
		main := clusters[maincluster]
		return main, append(clusters[:maincluster], clusters[maincluster+1:]...)
	}
	return clusters[0], []cluster{}
}

// buildClustersLists build a list of independant clusters
// we could have cluster partially overlapping in case of inconsistent cluster view
func buildClustersLists(infos *redis.ClusterInfos) []cluster {
	clusters := []cluster{}
	for _, nodeinfos := range infos.Infos {
		if nodeinfos == nil || nodeinfos.Node == nil {
			continue
		}
		slice := append(nodeinfos.Friends, nodeinfos.Node)
		var c cluster
		// build list of addresses
		for _, node := range slice {
			if len(node.FailStatus) == 0 {
				c = append(c, node.IPPort())
			}
		}
		// check if this cluster overlap with another
		overlap := false
		for _, node := range c {
			if findInCluster(node, clusters) {
				overlap = true
				break
			}
		}
		// if this is a new cluster, add it
		if !overlap {
			clusters = append(clusters, c)
		}
	}
	return clusters
}

func findInCluster(addr string, clusters []cluster) bool {
	for _, c := range clusters {
		for _, nodeAddr := range c {
			if addr == nodeAddr {
				return true
			}
		}
	}
	return false
}
