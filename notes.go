package main

import (
	"github.com/rancher/kontainer-engine/types"
)

// Driver defines the interface that each driver plugin should implement
type DriverInterface types.Driver

type GoogleKubernetesEngineConfig struct {
	// ProjectID is the ID of your project to use when creating a cluster
	ProjectID string `json:"projectId,omitempty" norman:"required"`
	// The zone to launch the cluster
	Zone string `json:"zone,omitempty" norman:"required"`
	// The IP address range of the container pods
	ClusterIpv4Cidr string `json:"clusterIpv4Cidr,omitempty"`
	// An optional description of this cluster
	Description string `json:"description,omitempty"`
	// The number of nodes in this cluster
	NodeCount int64 `json:"nodeCount,omitempty" norman:"required"`
	// Size of the disk attached to each node
	DiskSizeGb int64 `json:"diskSizeGb,omitempty"`
	// The name of a Google Compute Engine
	MachineType string `json:"machineType,omitempty"`
	// Node kubernetes version
	NodeVersion string `json:"nodeVersion,omitempty"`
	// the master kubernetes version
	MasterVersion string `json:"masterVersion,omitempty"`
	// The map of Kubernetes labels (key/value pairs) to be applied
	// to each node.
	Labels map[string]string `json:"labels,omitempty"`
	// The content of the credential file(key.json)
	Credential string `json:"credential,omitempty" norman:"required,type=password"`
	// Enable alpha feature
	EnableAlphaFeature bool `json:"enableAlphaFeature,omitempty"`
	// Configuration for the HTTP (L7) load balancing controller addon
	EnableHTTPLoadBalancing *bool `json:"enableHttpLoadBalancing,omitempty" norman:"default=true"`
	// Configuration for the horizontal pod autoscaling feature, which increases or decreases the number of replica pods a replication controller has based on the resource usage of the existing pods
	EnableHorizontalPodAutoscaling *bool `json:"enableHorizontalPodAutoscaling,omitempty" norman:"default=true"`
	// Configuration for the Kubernetes Dashboard
	EnableKubernetesDashboard bool `json:"enableKubernetesDashboard,omitempty"`
	// Configuration for NetworkPolicy
	EnableNetworkPolicyConfig *bool `json:"enableNetworkPolicyConfig,omitempty" norman:"default=true"`
	// The list of Google Compute Engine locations in which the cluster's nodes should be located
	Locations []string `json:"locations,omitempty"`
	// Image Type
	ImageType string `json:"imageType,omitempty"`
	// Network
	Network string `json:"network,omitempty"`
	// Sub Network
	SubNetwork string `json:"subNetwork,omitempty"`
	// Configuration for LegacyAbac
	EnableLegacyAbac            bool   `json:"enableLegacyAbac,omitempty"`
	EnableStackdriverLogging    *bool  `json:"enableStackdriverLogging,omitempty" norman:"default=true"`
	EnableStackdriverMonitoring *bool  `json:"enableStackdriverMonitoring,omitempty" norman:"default=true"`
	MaintenanceWindow           string `json:"maintenanceWindow"`
}

type AliyunKubernetesEngineConfig struct {
	//Common fields
	DisableRollback          bool   `json:"disableRollback,omitempty"`
	Name                     string `json:"name,omitempty"`
	ClusterType              string `json:"clusterYype,omitempty"`
	TimeoutMins              int    `json:"timeoutMins,omitempty"`
	RegionID                 string `json:"regionId,omitempty"`
	VpcID                    string `json:"vpcId,omitempty"`
	ZoneID                   string `json:"zoneId,omitempty"`
	VswitchID                string `json:"vswitchId,omitempty"`
	ContainerCidr            string `json:"containerCidr,omitempty"`
	ServiceCidr              string `json:"serviceCidr,omitempty"`
	CloudMonitorFlags        bool   `json:"cloudMonitorFlags,omitempty"`
	LoginPassword            string `json:"loginPassword,omitempty"`
	KeyPair                  string `json:"keyPair,omitempty"`
	WorkerInstanceChargeType string `json:"workerInstanceChargeType,omitempty"`
	WorkerPeriod             int    `json:"workerPeriod,omitempty"`
	WorkerPeriodUnit         string `json:"workerPeriodUnit,omitempty"`
	WorkerAutoRenew          bool   `json:"workerAutoRenew,omitempty"`
	WorkerAutoRenewPeriod    int    `json:"workerAutoRenewPeriod,omitempty"`
	WorkerInstanceType       string `json:"workerInstanceType,omitempty"`
	WorkerSystemDiskCategory string `json:"workerSystemDiskCategory,omitempty"`
	WorkerSystemDiskSize     int    `json:"workerSystemDiskSize,omitempty"`
	WorkerDataDisk           bool   `json:"workerDataDisk,omitempty"`
	WorkerDataDiskCategory   string `json:"workerDataDiskCategory,omitempty"`
	WorkerDataDiskSize       int    `json:"workerDataDiskSize,omitempty"`
	NumOfNodes               int    `json:"numOfNodes,omitempty"`
	SnatEntry                bool   `json:"snatEntry,omitempty"`

	//not managed Kubernetes fields
	//是否开放公网SSH登录
	SshFlags                 bool   `json:"sshFlags,omitempty"`
	MasterInstanceChangeType string `json:"masterInstanceChangeType,omitempty"`
	MasterPeriod             int    `json:"masterPeriod,omitempty"`
	MasterPeriodUnit         string `json:"masterPeriodUnit,omitempty"`
	MasterAutoRenew          bool   `json:"masterAutoRenew,omitempty"`
	MasterAutoRenewPeriod    int    `json:"masterAutoRenewPeriod,omitempty"`
	MasterInstanceType       string `json:"masterInstanceType,omitempty"`
	MasterSystemDiskCategory string `json:"masterSystemDiskCategory,omitempty"`
	MasterSystemDiskSize     int    `json:"masterSystemDiskSize,omitempty"`
	MasterDataDisk           bool   `json:"masterDataDisk,omitempty"`
	MasterDataDiskCategory   string `json:"masterDataDiskCategory,omitempty"`
	MasterDataDiskSize       int    `json:"masterDataDiskSize,omitempty"`
	PublicSlb                bool   `json:"publicSlb,omitempty"`

	//multi az type options
	MultiAz             bool   `json:"multiAz,omitempty"`
	VswitchIdA          string `json:"vswitchIdA,omitempty"`
	VswitchIdB          string `json:"vswitchIdB,omitempty"`
	VswitchIdC          string `json:"vswitchIdC,omitempty"`
	MasterInstanceTypeA string `json:"masterInstanceTypeA,omitempty"`
	MasterInstanceTypeB string `json:"masterInstanceTypeB,omitempty"`
	MasterInstanceTypeC string `json:"masterInstanceTypeC,omitempty"`
	WorkerInstanceTypeA string `json:"workerInstanceTypeA,omitempty"`
	WorkerInstanceTypeB string `json:"workerInstanceTypeB,omitempty"`
	WorkerInstanceTypeC string `json:"workerInstanceTypeC,omitempty"`
	NumOfNodesA         int    `json:"numOfNodesA,omitempty"`
	NumOfNodesB         int    `json:"numOfNodesB,omitempty"`
	NumOfNodesC         int    `json:"numOfNodesC,omitempty"`
}
