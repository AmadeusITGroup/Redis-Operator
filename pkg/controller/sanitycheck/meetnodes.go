package sanitycheck

import (
	"fmt"

	"github.com/golang/glog"

	"github.com/amadeusitgroup/redis-operator/pkg/redis"
)

// FixNodesNotMeet Use to fix Redis nodes that didn't them.
func FixNodesNotMeet(admin redis.AdminInterface, infos *redis.ClusterInfos, dryRun bool) (bool, error) {
	return fixNodesNotMeetFunc(admin, infos, nodesMeet, dryRun)
}

type nodesMeetFunc func(admin redis.AdminInterface, node1, node2 *redis.Node) error

func fixNodesNotMeetFunc(admin redis.AdminInterface, infos *redis.ClusterInfos, meetFunc nodesMeetFunc, dryRun bool) (bool, error) {
	if infos == nil || infos.Infos == nil {
		return false, nil
	}
	var globalErr error
	doneAnAction := false

	for _, node1 := range infos.Infos {
		for _, node2 := range infos.Infos {
			if node1.Node.ID == node2.Node.ID {
				continue
			}
			found := false
			for _, friend1 := range node1.Friends {
				if node2.Node.ID == friend1.ID {
					found = true
					break
				}
			}

			if !found {
				var err error
				if !dryRun {
					err = meetFunc(admin, node1.Node, node2.Node)
				}
				if err != nil {
					glog.Errorf("[SanityChecks] meet failed %s %s, err:%v", node1.Node.ID, node2.Node.ID, err)
					globalErr = err
				} else {
					doneAnAction = true
				}
			}
		}
	}

	return doneAnAction, globalErr
}

func nodesMeet(admin redis.AdminInterface, node1, node2 *redis.Node) error {
	glog.V(2).Infof("[Sanity] Cluster Meet, id1:%s id2:%s", node1.ID, node2.ID)
	c, err := admin.Connections().Get(node1.IPPort())
	if err != nil {
		return fmt.Errorf("[Sanity] unable get connection for node:%s, err:%v", node1.IPPort(), err)
	}
	resp := c.Cmd("CLUSTER", "MEET", node2.IP, node2.Port)
	if resp.Err != nil {
		return fmt.Errorf("[Sanity] unable to execute the cluster meet command, err:%v", resp.Err)
	}

	return nil
}
