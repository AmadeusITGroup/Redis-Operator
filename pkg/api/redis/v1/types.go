package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kapiv1 "k8s.io/api/core/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RedisCluster represents a Redis Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RedisCluster struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired RedisCluster specification
	Spec RedisClusterSpec `json:"spec,omitempty"`

	// Status represents the current RedisCluster status
	Status RedisClusterStatus `json:"status,omitempty"`
}

// RedisClusterList implements list of RedisCluster.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RedisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of RedisCluster
	Items []RedisCluster `json:"items"`
}

// RedisClusterSpec contains RedisCluster specification
type RedisClusterSpec struct {
	NumberOfMaster    *int32 `json:"numberOfMaster,omitempty"`
	ReplicationFactor *int32 `json:"replicationFactor,omitempty"`

	// ServiceName name used to create the Kubernetes Service that reference the Redis Cluster nodes.
	// if ServiceName is empty, the RedisCluster.Name will be use for creating the service.
	ServiceName string `json:"serviceName,omitempty"`

	// PodTemplate contains the pod specificaton that should run the redis-server process
	PodTemplate *kapiv1.PodTemplateSpec `json:"podTemplate,omitempty"`

	// Labels for created redis-cluster (deployment, rs, pod) (if any)
	AdditionalLabels map[string]string `json:"AdditionalLabels,omitempty"`
}

// RedisClusterStatus contains RedisCluster status
type RedisClusterStatus struct {
	// Conditions represent the latest available observations of an object's current state.
	Conditions []RedisClusterCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status kapiv1.ConditionStatus `json:"status"`
	// StartTime represents time when the workflow was acknowledged by the Workflow controller
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	// StartTime doesn't consider startime of `ExternalReference`
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// (brief) reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
	// Cluster a view of the current RedisCluster
	Cluster RedisClusterClusterStatus
}

// RedisClusterCondition represent the condition of the RedisCluster
type RedisClusterCondition struct {
	// Type of workflow condition
	Type RedisClusterConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status kapiv1.ConditionStatus `json:"status"`
	// Last time the condition was checked.
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transited from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
}

// RedisClusterClusterStatus represent the Redis Cluster status
type RedisClusterClusterStatus struct {
	Status               ClusterStatus `json:"status"`
	NumberOfMaster       int32         `json:"numberOfMaster,omitempty"`
	MinReplicationFactor int32         `json:"minReplicationFactor,omitempty"`
	MaxReplicationFactor int32         `json:"maxReplicationFactor,omitempty"`

	NodesPlacement NodesPlacementInfo `json:"nodesPlacementInfo,omitempty"`

	// In theory, we always have NbPods > NbRedisRunning > NbPodsReady
	NbPods         int32 `json:"nbPods,omitempty"`
	NbPodsReady    int32 `json:"nbPodsReady,omitempty"`
	NbRedisRunning int32 `json:"nbRedisNodesRunning,omitempty"`

	Nodes []RedisClusterNode `json:"nodes"`
}

func (s RedisClusterClusterStatus) String() string {
	output := ""
	output += fmt.Sprintf("status:%s\n", s.Status)
	output += fmt.Sprintf("NumberOfMaster:%d\n", s.NumberOfMaster)
	output += fmt.Sprintf("MinReplicationFactor:%d\n", s.MinReplicationFactor)
	output += fmt.Sprintf("MaxReplicationFactor:%d\n", s.MaxReplicationFactor)
	output += fmt.Sprintf("NodesPlacement:%s\n\n", s.NodesPlacement)
	output += fmt.Sprintf("NbPods:%d\n", s.NbPods)
	output += fmt.Sprintf("NbPodsReady:%d\n", s.NbPodsReady)
	output += fmt.Sprintf("NbRedisRunning:%d\n\n", s.NbRedisRunning)

	output += fmt.Sprintf("Nodes (%d): %s\n", len(s.Nodes), s.Nodes)

	return output
}

// NodesPlacementInfo Redis Nodes placement mode information
type NodesPlacementInfo string

const (
	// NodesPlacementInfoBestEffort the cluster nodes placement is in best effort,
	// it means you can have 2 masters (or more) on the same VM.
	NodesPlacementInfoBestEffort NodesPlacementInfo = "BestEffort"
	// NodesPlacementInfoOptimal the cluster nodes placement is optimal,
	// it means on master by VM
	NodesPlacementInfoOptimal NodesPlacementInfo = "Optimal"
)

// RedisClusterNode represent a RedisCluster Node
type RedisClusterNode struct {
	ID        string               `json:"id"`
	Role      RedisClusterNodeRole `json:"role"`
	IP        string               `json:"ip"`
	Port      string               `json:"port"`
	Slots     []string             `json:"slots,omitempty"`
	MasterRef string               `json:"masterRef,omitempty"`
	PodName   string               `json:"podName"`
	Pod       *kapiv1.Pod          `json:"-"`
}

func (n RedisClusterNode) String() string {
	if n.Role != RedisClusterNodeRoleSlave {
		return fmt.Sprintf("(Master:%s, Addr:%s:%s, PodName:%s, Slots:%v)", n.ID, n.IP, n.Port, n.PodName, n.Slots)
	}
	return fmt.Sprintf("(Slave:%s, Addr:%s:%s, PodName:%s, MasterRef:%s)", n.ID, n.IP, n.Port, n.PodName, n.MasterRef)
}

// RedisClusterConditionType is the type of RedisClusterCondition
type RedisClusterConditionType string

const (
	// RedisClusterOK means the RedisCluster is in a good shape
	RedisClusterOK RedisClusterConditionType = "ClusterOK"
	// RedisClusterScaling means the RedisCluster is currenlty in a scaling stage
	RedisClusterScaling RedisClusterConditionType = "Scaling"
	// RedisClusterRebalancing means the RedisCluster is currenlty rebalancing slots and keys
	RedisClusterRebalancing RedisClusterConditionType = "Rebalancing"
	// RedisClusterRollingUpdate means the RedisCluster is currenlty performing a rolling update of its nodes
	RedisClusterRollingUpdate RedisClusterConditionType = "RollingUpdate"
)

// RedisClusterNodeRole RedisCluster Node Role type
type RedisClusterNodeRole string

const (
	// RedisClusterNodeRoleMaster RedisCluster Master node role
	RedisClusterNodeRoleMaster RedisClusterNodeRole = "Master"
	// RedisClusterNodeRoleSlave RedisCluster Master node role
	RedisClusterNodeRoleSlave RedisClusterNodeRole = "Slave"
	// RedisClusterNodeRoleNone None node role
	RedisClusterNodeRoleNone RedisClusterNodeRole = "None"
)

// ClusterStatus Redis Cluster status
type ClusterStatus string

const (
	// ClusterStatusOK ClusterStatus OK
	ClusterStatusOK ClusterStatus = "OK"
	// ClusterStatusKO ClusterStatus KO
	ClusterStatusKO ClusterStatus = "KO"
	// ClusterStatusScaling ClusterStatus Scaling
	ClusterStatusScaling ClusterStatus = "Scaling"
	// ClusterStatusCalculatingRebalancing ClusterStatus Rebalancing
	ClusterStatusCalculatingRebalancing ClusterStatus = "Calculating Rebalancing"
	// ClusterStatusRebalancing ClusterStatus Rebalancing
	ClusterStatusRebalancing ClusterStatus = "Rebalancing"
	// ClusterStatusRollingUpdate ClusterStatus RollingUpdate
	ClusterStatusRollingUpdate ClusterStatus = "RollingUpdate"
)
