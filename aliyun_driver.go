package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/rancher/kontainer-engine/drivers/options"
	"github.com/rancher/kontainer-engine/drivers/util"
	"github.com/rancher/kontainer-engine/types"
	"github.com/rancher/rke/log"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	runningStatus = "running"
	failedStatus  = "failed"
	retries       = 5
	pollInterval  = 30
)

var EnvMutex sync.Mutex

func init() {
	// GMT IANA timezone data
	gmtTzData := []byte{84, 90, 105, 102, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 71, 77, 84, 0, 0, 0, 10, 71, 77, 84, 48, 10}
	utils.LoadLocationFromTZData = time.LoadLocationFromTZData
	utils.TZData = gmtTzData
}

// Driver defines the struct of aliyun driver
type Driver struct {
	driverCapabilities types.Capabilities
	k8sCapabilities    types.K8SCapabilities
}

type state struct {
	// The displays name of the cluster
	Name string `json:"name,omitempty"`
	//Common fields
	ClusterID                string `json:"cluster_id,omitempty"`
	AccessKeyID              string `json:"accessKeyId,omitempty"`
	AccessKeySecret          string `json:"accessKeySecret,omitempty"`
	DisableRollback          bool   `json:"disable_rollback,omitempty"`
	ClusterType              string `json:"cluster_type,omitempty"`
	KubernetesVersion        string `json:"kubernetes_version,omitempty"`
	TimeoutMins              int64  `json:"timeout_mins,omitempty"`
	RegionID                 string `json:"region_id,omitempty"`
	VpcID                    string `json:"vpcid,omitempty"`
	ZoneID                   string `json:"zoneid,omitempty"`
	VswitchID                string `json:"vswitchid,omitempty"`
	ContainerCidr            string `json:"container_cidr,omitempty"`
	ServiceCidr              string `json:"service_cidr,omitempty"`
	CloudMonitorFlags        bool   `json:"cloud_monitor_flags,omitempty"`
	LoginPassword            string `json:"login_password,omitempty"`
	KeyPair                  string `json:"key_pair,omitempty"`
	WorkerInstanceChargeType string `json:"worker_instance_charge_type,omitempty"`
	WorkerPeriod             int64  `json:"worker_period,omitempty"`
	WorkerPeriodUnit         string `json:"worker_period_unit,omitempty"`
	WorkerAutoRenew          bool   `json:"worker_auto_renew,omitempty"`
	WorkerAutoRenewPeriod    int64  `json:"worker_auto_renew_period,omitempty"`
	WorkerInstanceType       string `json:"worker_instance_type,omitempty"`
	WorkerSystemDiskCategory string `json:"worker_system_disk_category,omitempty"`
	WorkerSystemDiskSize     int64  `json:"worker_system_disk_size,omitempty"`
	WorkerDataDisk           bool   `json:"worker_data_disk,omitempty"`
	WorkerDataDiskCategory   string `json:"worker_data_disk_category,omitempty"`
	WorkerDataDiskSize       int64  `json:"worker_data_disk_size,omitempty"`
	NumOfNodes               int64  `json:"num_of_nodes,omitempty"`
	SnatEntry                bool   `json:"snat_entry,omitempty"`
	// non-managed Kubernetes fields
	SSHFlags                 bool   `json:"ssh_flags,omitempty"`
	MasterInstanceChargeType string `json:"master_instance_charge_type,omitempty"`
	MasterPeriod             int64  `json:"master_period,omitempty"`
	MasterPeriodUnit         string `json:"master_period_unit,omitempty"`
	MasterAutoRenew          bool   `json:"master_auto_renew,omitempty"`
	MasterAutoRenewPeriod    int64  `json:"master_auto_renew_period,omitempty"`
	MasterInstanceType       string `json:"master_instance_type,omitempty"`
	MasterSystemDiskCategory string `json:"master_system_disk_category,omitempty"`
	MasterSystemDiskSize     int64  `json:"master_system_disk_size,omitempty"`
	MasterDataDisk           bool   `json:"master_data_disk,omitempty"`
	MasterDataDiskCategory   string `json:"master_data_disk_category,omitempty"`
	MasterDataDiskSize       int64  `json:"master_data_disk_size,omitempty"`
	PublicSlb                bool   `json:"public_slb,omitempty"`
	// multi-az kubernetes options
	MultiAz             bool   `json:"multi_az,omitempty"`
	VswitchIDA          string `json:"vswitch_id_a,omitempty"`
	VswitchIDB          string `json:"vswitch_id_b,omitempty"`
	VswitchIDC          string `json:"vswitch_id_c,omitempty"`
	MasterInstanceTypeA string `json:"master_instance_type_a,omitempty"`
	MasterInstanceTypeB string `json:"master_instance_type_b,omitempty"`
	MasterInstanceTypeC string `json:"master_instance_type_c,omitempty"`
	WorkerInstanceTypeA string `json:"worker_instance_type_a,omitempty"`
	WorkerInstanceTypeB string `json:"worker_instance_type_b,omitempty"`
	WorkerInstanceTypeC string `json:"worker_instance_type_c,omitempty"`
	NumOfNodesA         int64  `json:"num_of_nodes_a,omitempty"`
	NumOfNodesB         int64  `json:"num_of_nodes_b,omitempty"`
	NumOfNodesC         int64  `json:"num_of_nodes_c,omitempty"`

	// cluster info
	ClusterInfo types.ClusterInfo
}

type clusterGetResponse struct {
	State          string `json:"state,omitempty"`
	Size           int64  `json:"size,omitempty"`
	CurrentVersion string `json:"current_version,omitempty"`
}

type clusterCreateResponse struct {
	ClusterID string `json:"cluster_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
}

type clusterUserConfig struct {
	Config string
}

type clusterCerts struct {
	Ca   string `json:"ca"`
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

type clusterLog struct {
	ID         int64
	ClusterID  string `json:"cluster_id"`
	ClusterLog string `json:"cluster_log"`
	LogLevel   string `json:"log_level"`
	Created    string `json:"created"`
	Updated    string `json:"updated"`
}

func NewDriver() types.Driver {
	driver := &Driver{
		driverCapabilities: types.Capabilities{
			Capabilities: make(map[int64]bool),
		},
	}

	driver.driverCapabilities.AddCapability(types.GetVersionCapability)
	driver.driverCapabilities.AddCapability(types.GetClusterSizeCapability)
	driver.driverCapabilities.AddCapability(types.SetClusterSizeCapability)

	return driver
}

// GetDriverCreateOptions implements driver interface
func (d *Driver) GetDriverCreateOptions(ctx context.Context) (*types.DriverFlags, error) {
	driverFlag := types.DriverFlags{
		Options: make(map[string]*types.Flag),
	}
	driverFlag.Options["name"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the name of the cluster",
	}
	driverFlag.Options["display-name"] = &types.Flag{
		Type:  types.StringType,
		Usage: "the display name of the cluster",
	}
	driverFlag.Options["access-key-id"] = &types.Flag{
		Type:     types.StringType,
		Usage:    "AcessKeyId",
		Password: true,
	}
	driverFlag.Options["access-key-secret"] = &types.Flag{
		Type:     types.StringType,
		Usage:    "AccessKeySecret",
		Password: true,
	}
	driverFlag.Options["disable-rollback"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Whether or not to roll back if the cluster fails to be created",
	}
	driverFlag.Options["cluster-type"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Cluster type, Kubernetes or ManagedKubernetes",
		Default: &types.Default{
			DefaultString: "ManagedKubernetes",
		},
	}
	driverFlag.Options["kubernetes-version"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Kubernetes version",
	}
	driverFlag.Options["timeout-mins"] = &types.Flag{
		Type:  types.IntType,
		Usage: "The timeout (in minutes) for creating the cluster resource stack.",
	}
	driverFlag.Options["region-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The ID of the region in which the cluster resides",
	}
	driverFlag.Options["zone-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The zone of the region in which the cluster resides",
	}
	driverFlag.Options["vpc-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The VPC ID, which can be empty. If left empty, the system automatically creates a VPC.",
	}
	driverFlag.Options["vswitch-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "The VSwitch ID, which can be empty. If left empty, the system automatically creates a VSwitch.",
	}
	driverFlag.Options["container-cidr"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Pod CIDR",
	}
	driverFlag.Options["service-cidr"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Service CIDR",
	}
	driverFlag.Options["worker-instance-charge-type"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Worker node payment type PrePaid|PostPaid",
		Default: &types.Default{
			DefaultString: "PostPaid",
		},
	}
	driverFlag.Options["worker-period-unit"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Subscription unit, which includes month and year, and takes effect only for the prepaid type.",
	}
	driverFlag.Options["worker-period"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Subscription period, which takes effect only for the prepaid type",
	}
	driverFlag.Options["worker-auto-renew"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Worker node auto renew",
	}
	driverFlag.Options["worker-auto-renew-period"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Worker node renew period",
	}
	driverFlag.Options["worker-data-disk"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Whether or not to mount data disks",
	}
	driverFlag.Options["worker-data-disk-category"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Data disk category",
	}
	driverFlag.Options["worker-data-disk-size"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Data disk size",
	}
	driverFlag.Options["worker-instance-type"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Instance type of worker nodes",
	}
	driverFlag.Options["worker-system-disk-category"] = &types.Flag{
		Type:  types.StringType,
		Usage: "System disk type of worker nodes",
	}
	driverFlag.Options["worker-system-disk-size"] = &types.Flag{
		Type:  types.IntType,
		Usage: "System disk size of worker nodes",
	}
	driverFlag.Options["login-password"] = &types.Flag{
		Type:     types.StringType,
		Usage:    "Password used to log on to the node by using SSH.",
		Password: true,
	}
	driverFlag.Options["key-pair"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Key pair name used to log on to the node by using SSH.",
	}
	driverFlag.Options["num-of-nodes"] = &types.Flag{
		Type:  types.IntType,
		Usage: "number of worker nodes, the range is [0,300]",
	}
	driverFlag.Options["snat-entry"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Whether or not to configure the SNATEntry",
		Default: &types.Default{
			DefaultBool: true,
		},
	}
	driverFlag.Options["cloud-monitor-flags"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Whether or not to install the cloud monitoring plug-in",
	}
	driverFlag.Options["ssh-flags"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Whether or not to enable SSH access for Internet",
	}
	driverFlag.Options["master-instance-charge-type"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Master node payment type PrePaid|PostPaid",
	}
	driverFlag.Options["master-period"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Subscription period, which takes effect only for the prepaid type",
	}
	driverFlag.Options["master-period-unit"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Subscription unit, which includes month and year, and takes effect only for the prepaid type.",
	}
	driverFlag.Options["master-auto-renew"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Master node auto renew",
	}
	driverFlag.Options["master-auto-renew-period"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Master node renew period",
	}
	driverFlag.Options["master-instance-type"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Instance type of master nodes",
	}
	driverFlag.Options["master-system-disk-category"] = &types.Flag{
		Type:  types.StringType,
		Usage: "System disk type of master nodes",
	}
	driverFlag.Options["master-system-disk-size"] = &types.Flag{
		Type:  types.IntType,
		Usage: "System disk size of master nodes",
	}
	driverFlag.Options["master-data-disk"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Whether or not to mount data disks",
	}
	driverFlag.Options["master-data-disk-category"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Data disk category",
	}
	driverFlag.Options["master-data-disk-size"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Data disk size",
	}
	driverFlag.Options["public-slb"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Whether or not to create SLB to the API server",
	}
	driverFlag.Options["multi-az"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "Whether or not to set up master nodes in multiple available zones",
	}
	driverFlag.Options["vswitch-id-a"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Vswitch id of zone A",
	}
	driverFlag.Options["vswitch-id-b"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Vswitch id of zone B",
	}
	driverFlag.Options["vswitch-id-c"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Vswitch id of zone C",
	}
	driverFlag.Options["master-instance-type-a"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Instance type of the master node in zone A",
	}
	driverFlag.Options["master-instance-type-b"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Instance type of the master node in zone B",
	}
	driverFlag.Options["master-instance-type-c"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Instance type of the master node in zone C",
	}
	driverFlag.Options["worker-instance-type-a"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Instance type of the worker nodes in zone A",
	}
	driverFlag.Options["worker-instance-type-b"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Instance type of the worker nodes in zone B",
	}
	driverFlag.Options["worker-instance-type-c"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Instance type of the worker nodes in zone C",
	}
	driverFlag.Options["num-of-nodes-a"] = &types.Flag{
		Type:  types.IntType,
		Usage: "number of worker nodes in zone A",
	}
	driverFlag.Options["num-of-nodes-b"] = &types.Flag{
		Type:  types.IntType,
		Usage: "number of worker nodes in zone B",
	}
	driverFlag.Options["num-of-nodes-c"] = &types.Flag{
		Type:  types.IntType,
		Usage: "number of worker nodes in zone C",
	}

	return &driverFlag, nil
}

// GetDriverUpdateOptions implements driver interface
func (d *Driver) GetDriverUpdateOptions(ctx context.Context) (*types.DriverFlags, error) {
	driverFlag := types.DriverFlags{
		Options: make(map[string]*types.Flag),
	}
	driverFlag.Options["num-of-nodes"] = &types.Flag{
		Type:  types.IntType,
		Usage: "The node number for your cluster to update. 0 means no updates",
	}
	return &driverFlag, nil
}

// SetDriverOptions implements driver interface
func getStateFromOpts(driverOptions *types.DriverOptions) (*state, error) {
	d := &state{
		ClusterInfo: types.ClusterInfo{
			Metadata: map[string]string{},
		},
	}
	d.Name = options.GetValueFromDriverOptions(driverOptions, types.StringType, "display-name", "displayName").(string)
	d.AccessKeyID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "access-key-id", "accessKeyId").(string)
	d.AccessKeySecret = options.GetValueFromDriverOptions(driverOptions, types.StringType, "access-key-secret", "accessKeySecret").(string)
	d.DisableRollback = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "disable-rollback", "disableRollback").(bool)
	d.ClusterType = options.GetValueFromDriverOptions(driverOptions, types.StringType, "cluster-type", "clusterType").(string)
	d.KubernetesVersion = options.GetValueFromDriverOptions(driverOptions, types.StringType, "kubernetes-version", "kubernetesVersion").(string)
	d.TimeoutMins = options.GetValueFromDriverOptions(driverOptions, types.IntType, "timeout-mins", "timeoutMins").(int64)
	d.RegionID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "region-id", "regionId").(string)
	d.VpcID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "vpc-id", "vpcId").(string)
	d.ZoneID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "zone-id", "zoneId").(string)
	d.VswitchID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "vswitch-id", "vswitchId").(string)
	d.ContainerCidr = options.GetValueFromDriverOptions(driverOptions, types.StringType, "container-cidr", "containerCidr").(string)
	d.ServiceCidr = options.GetValueFromDriverOptions(driverOptions, types.StringType, "service-cidr", "serviceCidr").(string)
	d.CloudMonitorFlags = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "cloud-monitor-flags", "cloudMonitorFlags").(bool)
	d.LoginPassword = options.GetValueFromDriverOptions(driverOptions, types.StringType, "login-password", "loginPassword").(string)
	d.KeyPair = options.GetValueFromDriverOptions(driverOptions, types.StringType, "key-pair", "keyPair").(string)
	d.WorkerInstanceChargeType = options.GetValueFromDriverOptions(driverOptions, types.StringType, "worker-instance-charge-type", "workerInstanceChargeType").(string)
	d.WorkerPeriod = options.GetValueFromDriverOptions(driverOptions, types.IntType, "worker-period", "workerPeriod").(int64)
	d.WorkerPeriodUnit = options.GetValueFromDriverOptions(driverOptions, types.StringType, "worker-period-unit", "workerPeriodUnit").(string)
	d.WorkerAutoRenew = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "worker-auto-renew", "workerAutoRenew").(bool)
	d.WorkerAutoRenewPeriod = options.GetValueFromDriverOptions(driverOptions, types.IntType, "worker-auto-renew-period", "workerAutoRenewPeriod").(int64)
	d.WorkerInstanceType = options.GetValueFromDriverOptions(driverOptions, types.StringType, "worker-instance-type", "workerInstanceType").(string)
	d.WorkerSystemDiskCategory = options.GetValueFromDriverOptions(driverOptions, types.StringType, "worker-system-disk-category", "workerSystemDiskCategory").(string)
	d.WorkerSystemDiskSize = options.GetValueFromDriverOptions(driverOptions, types.IntType, "worker-system-disk-size", "workerSystemDiskSize").(int64)
	d.WorkerDataDisk = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "worker-data-disk", "workerDataDisk").(bool)
	d.WorkerDataDiskCategory = options.GetValueFromDriverOptions(driverOptions, types.StringType, "worker-data-disk-category", "workerDataDiskCategory").(string)
	d.WorkerDataDiskSize = options.GetValueFromDriverOptions(driverOptions, types.IntType, "worker-data-disk-size", "workerDataDiskSize").(int64)
	d.NumOfNodes = options.GetValueFromDriverOptions(driverOptions, types.IntType, "num-of-nodes", "numOfNodes").(int64)
	d.SnatEntry = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "snat-entry", "snatEntry").(bool)

	d.SSHFlags = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "ssh-flags", "sshFlags").(bool)
	d.MasterInstanceChargeType = options.GetValueFromDriverOptions(driverOptions, types.StringType, "master-instance-charge-type", "masterInstanceChargeType").(string)
	d.MasterPeriod = options.GetValueFromDriverOptions(driverOptions, types.IntType, "master-period", "masterPeriod").(int64)
	d.MasterPeriodUnit = options.GetValueFromDriverOptions(driverOptions, types.StringType, "master-period-unit", "masterPeriodUnit").(string)
	d.MasterAutoRenew = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "master-auto-renew", "masterAutoRenew").(bool)
	d.MasterAutoRenewPeriod = options.GetValueFromDriverOptions(driverOptions, types.IntType, "master-auto-renew-period", "masterAutoRenewPeriod").(int64)
	d.MasterInstanceType = options.GetValueFromDriverOptions(driverOptions, types.StringType, "master-instance-type", "masterInstanceType").(string)
	d.MasterSystemDiskCategory = options.GetValueFromDriverOptions(driverOptions, types.StringType, "master-system-disk-category", "masterSystemDiskCategory").(string)
	d.MasterSystemDiskSize = options.GetValueFromDriverOptions(driverOptions, types.IntType, "master-system-disk-size", "masterSystemDiskSize").(int64)
	d.MasterDataDisk = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "master-data-disk", "masterDataDisk").(bool)
	d.MasterDataDiskCategory = options.GetValueFromDriverOptions(driverOptions, types.StringType, "master-data-disk-category", "masterDataDiskCategory").(string)
	d.MasterDataDiskSize = options.GetValueFromDriverOptions(driverOptions, types.IntType, "master-data-disk-size", "masterDataDiskSize").(int64)
	d.PublicSlb = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "public-slb", "publicSlb").(bool)

	d.MultiAz = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "multi-az", "multiAz").(bool)
	d.VswitchIDA = options.GetValueFromDriverOptions(driverOptions, types.StringType, "vswitch-id-a", "VswitchIdA").(string)
	d.VswitchIDB = options.GetValueFromDriverOptions(driverOptions, types.StringType, "vswitch-id-b", "VswitchIdB").(string)
	d.VswitchIDC = options.GetValueFromDriverOptions(driverOptions, types.StringType, "vswitch-id-c", "VswitchIdC").(string)
	d.MasterInstanceTypeA = options.GetValueFromDriverOptions(driverOptions, types.StringType, "master-instance-type-a", "masterInstanceTypeA").(string)
	d.MasterInstanceTypeB = options.GetValueFromDriverOptions(driverOptions, types.StringType, "master-instance-type-b", "masterInstanceTypeB").(string)
	d.MasterInstanceTypeC = options.GetValueFromDriverOptions(driverOptions, types.StringType, "master-instance-type-c", "masterInstanceTypeC").(string)
	d.WorkerInstanceTypeA = options.GetValueFromDriverOptions(driverOptions, types.StringType, "worker-instance-type-a", "workerInstanceTypeA").(string)
	d.WorkerInstanceTypeB = options.GetValueFromDriverOptions(driverOptions, types.StringType, "worker-instance-type-b", "workerInstanceTypeB").(string)
	d.WorkerInstanceTypeC = options.GetValueFromDriverOptions(driverOptions, types.StringType, "worker-instance-type-c", "workerInstanceTypeC").(string)
	d.NumOfNodesA = options.GetValueFromDriverOptions(driverOptions, types.IntType, "num-of-nodes-a", "numOfNodesA").(int64)
	d.NumOfNodesB = options.GetValueFromDriverOptions(driverOptions, types.IntType, "num-of-nodes-b", "numOfNodesB").(int64)
	d.NumOfNodesC = options.GetValueFromDriverOptions(driverOptions, types.IntType, "num-of-nodes-c", "numOfNodesC").(int64)

	return d, d.validate()
}

func (s *state) validate() error {
	if s.Name == "" {
		return fmt.Errorf("cluster display name is required")
	} else if s.AccessKeyID == "" {
		return fmt.Errorf("access key id is required")
	} else if s.AccessKeySecret == "" {
		return fmt.Errorf("access key secret is required")
	} else if s.RegionID == "" {
		return fmt.Errorf("region id is required")
	} else if s.ZoneID == "" {
		return fmt.Errorf("zone id is required")
	} else if s.WorkerInstanceType == "" {
		return fmt.Errorf("worker instance type is required")
	} else if s.WorkerSystemDiskCategory == "" {
		return fmt.Errorf("worker system disk category is required")
	} else if s.WorkerSystemDiskSize <= 0 {
		return fmt.Errorf("worker system disk size is required")
	} else if s.LoginPassword == "" && s.KeyPair == "" {
		return fmt.Errorf("either login password or key pair name is needed")
	} else if s.NumOfNodes < 0 || s.NumOfNodes > 300 {
		return fmt.Errorf("number of nodes is required and supported range is [0,300]")
	} else if s.VpcID == "" && !s.SnatEntry {
		return fmt.Errorf("snat entry is required when vpc is auto created")
	} else if s.WorkerInstanceChargeType == "PrePaid" && s.WorkerPeriodUnit == "" {
		return fmt.Errorf("worker period unit is required for prepaid mode")
	}
	return nil
}

func (d *Driver) getAliyunServiceClient(ctx context.Context, state *state) (*cs.Client, error) {
	config := sdk.NewConfig().
		WithAutoRetry(false).
		WithTimeout(time.Minute).
		WithDebug(true)
	credential := &credentials.AccessKeyCredential{
		AccessKeyId:     state.AccessKeyID,
		AccessKeySecret: state.AccessKeySecret,
	}
	return cs.NewClientWithOptions(state.RegionID, config, credential)
}

func (d *Driver) waitAliyunCluster(ctx context.Context, svc *cs.Client, state *state) error {
	lastMsg := ""
	for {
		cluster, err := getCluster(svc, state)
		if err != nil {
			return err
		}
		status, err := getClusterLastMessage(svc, state)
		if err != nil {
			return err
		}
		if status != lastMsg {
			log.Infof(ctx, "provisioning cluster %s:%s", state.Name, status)
			lastMsg = status
		}

		if cluster.State == runningStatus {
			log.Infof(ctx, "Cluster %v is running", state.Name)
			return nil
		} else if cluster.State == failedStatus {
			return fmt.Errorf("aliyun failed to provision cluster: %s", status)
		}
		time.Sleep(time.Second * 15)
	}
}

func createCluster(svc *cs.Client, state *state) (*clusterCreateResponse, error) {
	request := NewCsAPIRequest("CreateCluster", requests.POST)
	request.PathPattern = "/clusters"
	content, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	request.SetContent(content)
	cluster := &clusterCreateResponse{}
	if err := ProcessRequest(svc, request, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func getCluster(svc *cs.Client, state *state) (*clusterGetResponse, error) {
	request := NewCsAPIRequest("DescribeClusterDetail", requests.GET)
	request.PathPattern = "/clusters/[ClusterId]"
	request.PathParams["ClusterId"] = state.ClusterID
	cluster := &clusterGetResponse{}
	if err := ProcessRequest(svc, request, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func putCluster(svc *cs.Client, state *state) error {
	m := make(map[string]interface{})
	m["disable_rollback"] = state.DisableRollback
	m["timeout_mins"] = state.TimeoutMins
	m["worker_instance_type"] = state.WorkerInstanceType
	m["login_password"] = state.LoginPassword
	m["num_of_nodes"] = state.NumOfNodes
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	request := NewCsAPIRequest("ScaleCluster", requests.PUT)
	request.PathPattern = "/clusters/[ClusterId]"
	request.PathParams["ClusterId"] = state.ClusterID
	request.SetContent(b)
	return ProcessRequest(svc, request, nil)
}

func deleteCluster(svc *cs.Client, state *state) error {
	request := NewCsAPIRequest("DeleteCluster", requests.DELETE)
	request.PathPattern = "/clusters/[ClusterId]"
	request.PathParams["ClusterId"] = state.ClusterID
	return ProcessRequest(svc, request, nil)
}

func getClusterUserConfig(svc *cs.Client, state *state) (*api.Config, error) {
	request := NewCsAPIRequest("DescribeClusterTokens", requests.GET)
	request.PathPattern = "/k8s/[ClusterId]/user_config"
	request.PathParams["ClusterId"] = state.ClusterID

	userConfig := &clusterUserConfig{}
	if err := ProcessRequest(svc, request, userConfig); err != nil {
		return nil, err
	}
	clientConfig, err := clientcmd.Load([]byte(userConfig.Config))
	if err != nil {
		return nil, err
	}
	return clientConfig, validateConfig(clientConfig)
}

func validateConfig(config *api.Config) error {
	if config == nil {
		return fmt.Errorf("get nil config")
	} else if config.Contexts[config.CurrentContext] == nil {
		return fmt.Errorf("invalid context in config")
	} else if config.Clusters[config.Contexts[config.CurrentContext].Cluster] == nil {
		return fmt.Errorf("invalid cluster in config")
	}
	return nil
}

func getClusterCerts(svc *cs.Client, state *state) (*clusterCerts, error) {
	request := NewCsAPIRequest("DescribeClusterCerts", requests.GET)
	request.PathPattern = "/clusters/[ClusterId]/certs"
	request.PathParams["ClusterId"] = state.ClusterID
	certs := &clusterCerts{}
	if err := ProcessRequest(svc, request, certs); err != nil {
		return nil, err
	}
	return certs, nil
}

func getClusterLastMessage(svc *cs.Client, state *state) (string, error) {
	request := NewCsAPIRequest("DescribeClusterLogs", requests.GET)
	request.PathPattern = "/clusters/[ClusterId]/logs"
	request.PathParams["ClusterId"] = state.ClusterID
	logs := []clusterLog{}
	if err := ProcessRequest(svc, request, &logs); err != nil {
		return "", err
	}
	if len(logs) <= 0 {
		return "", nil
	}
	lastMessage := logs[0].ClusterLog
	parts := strings.SplitN(logs[0].ClusterLog, "|", 2)
	if len(parts) == 2 {
		lastMessage = parts[1]
	}
	return lastMessage, nil
}

// Create implements driver interface
func (d *Driver) Create(ctx context.Context, opts *types.DriverOptions, _ *types.ClusterInfo) (*types.ClusterInfo, error) {
	state, err := getStateFromOpts(opts)
	if err != nil {
		return nil, err
	}

	svc, err := d.getAliyunServiceClient(ctx, state)
	if err != nil {
		return nil, err
	}

	cluster, err := createCluster(svc, state)
	if err != nil && !strings.Contains(err.Error(), "AlreadyExist") {
		return nil, err
	}
	if err == nil {
		state.ClusterID = cluster.ClusterID
	}

	if err := d.waitAliyunCluster(ctx, svc, state); err != nil {
		return nil, err
	}

	info := &types.ClusterInfo{}
	return info, storeState(info, state)
}

func storeState(info *types.ClusterInfo, state *state) error {
	bytes, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if info.Metadata == nil {
		info.Metadata = map[string]string{}
	}
	info.Metadata["state"] = string(bytes)
	return nil
}

func getState(info *types.ClusterInfo) (*state, error) {
	state := &state{}
	err := json.Unmarshal([]byte(info.Metadata["state"]), state)
	return state, err
}

// Update implements driver interface
func (d *Driver) Update(ctx context.Context, info *types.ClusterInfo, opts *types.DriverOptions) (*types.ClusterInfo, error) {
	logrus.Debug("update unimplemented")
	return info, nil
}

func (d *Driver) PostCheck(ctx context.Context, info *types.ClusterInfo) (*types.ClusterInfo, error) {
	state, err := getState(info)
	if err != nil {
		return nil, err
	}
	svc, err := d.getAliyunServiceClient(ctx, state)
	if err != nil {
		return nil, err
	}

	if err := d.waitAliyunCluster(ctx, svc, state); err != nil {
		return nil, err
	}
	cluster, err := getCluster(svc, state)
	if err != nil {
		return nil, err
	}
	userConfig, err := getClusterUserConfig(svc, state)
	if err != nil {
		return nil, err
	}
	certs, err := getClusterCerts(svc, state)
	if err != nil {
		return nil, err
	}
	currentContext := userConfig.Contexts[userConfig.CurrentContext]

	info.Endpoint = userConfig.Clusters[currentContext.Cluster].Server
	info.Version = cluster.CurrentVersion
	info.RootCaCertificate = base64.StdEncoding.EncodeToString([]byte(certs.Ca))
	info.ClientCertificate = base64.StdEncoding.EncodeToString([]byte(certs.Cert))
	info.ClientKey = base64.StdEncoding.EncodeToString([]byte(certs.Key))
	info.NodeCount = cluster.Size

	host := info.Endpoint
	if !strings.HasPrefix(host, "https://") {
		host = fmt.Sprintf("https://%s", host)
	}

	config := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   []byte(certs.Ca),
			KeyData:  []byte(certs.Key),
			CertData: []byte(certs.Cert),
		},
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %v", err)
	}

	failureCount := 0
	for {
		info.ServiceAccountToken, err = util.GenerateServiceAccountToken(clientset)

		if err == nil {
			logrus.Info("service account token generated successfully")
			break
		} else {
			if failureCount < retries {
				logrus.Infof("service account token generation failed, retries left: %v", retries-failureCount)
				failureCount = failureCount + 1

				time.Sleep(pollInterval * time.Second)
			} else {
				logrus.Error("retries exceeded, failing post-check")
				return nil, err
			}
		}
	}
	logrus.Info("post-check completed successfully")
	return info, nil
}

// Remove implements driver interface
func (d *Driver) Remove(ctx context.Context, info *types.ClusterInfo) error {
	state, err := getState(info)
	if err != nil {
		return err
	}
	svc, err := d.getAliyunServiceClient(ctx, state)
	if err != nil {
		return err
	}
	logrus.Debugf("Removing cluster %v from region %v, zone %v", state.Name, state.RegionID, state.ZoneID)
	if err := deleteCluster(svc, state); err != nil && !strings.Contains(err.Error(), "NotFound") {
		return err
	} else if err == nil {
		logrus.Debugf("Cluster %v delete is called.", state.Name)
	} else {
		logrus.Debugf("Cluster %s doesn't exist", state.Name)
	}
	return nil
}

func (d *Driver) GetClusterSize(ctx context.Context, info *types.ClusterInfo) (*types.NodeCount, error) {
	state, err := getState(info)
	if err != nil {
		return nil, err
	}
	svc, err := d.getAliyunServiceClient(ctx, state)
	if err != nil {
		return nil, err
	}
	cluster, err := getCluster(svc, state)
	if err != nil {
		return nil, err
	}
	return &types.NodeCount{Count: cluster.Size}, nil
}

func (d *Driver) GetVersion(ctx context.Context, info *types.ClusterInfo) (*types.KubernetesVersion, error) {
	state, err := getState(info)
	if err != nil {
		return nil, err
	}
	svc, err := d.getAliyunServiceClient(ctx, state)
	if err != nil {
		return nil, err
	}
	cluster, err := getCluster(svc, state)
	if err != nil {
		return nil, err
	}
	return &types.KubernetesVersion{Version: cluster.CurrentVersion}, nil
}

func (d *Driver) SetClusterSize(ctx context.Context, info *types.ClusterInfo, count *types.NodeCount) error {
	state, err := getState(info)
	if err != nil {
		return err
	}
	svc, err := d.getAliyunServiceClient(ctx, state)
	if err != nil {
		return err
	}
	state.NumOfNodes = count.GetCount()
	if err := putCluster(svc, state); err != nil {
		return err
	}
	if err := d.waitAliyunCluster(ctx, svc, state); err != nil {
		return err
	}
	logrus.Info("cluster size updated successfully")
	return nil
}

func (d *Driver) SetVersion(ctx context.Context, info *types.ClusterInfo, version *types.KubernetesVersion) error {
	logrus.Debug("setversion unimplemented")
	return nil
}

func (d *Driver) GetCapabilities(ctx context.Context) (*types.Capabilities, error) {
	return &d.driverCapabilities, nil
}

func (d *Driver) GetK8SCapabilities(ctx context.Context, opts *types.DriverOptions) (*types.K8SCapabilities, error) {
	return &types.K8SCapabilities{
		L4LoadBalancer: &types.LoadBalancerCapabilities{
			Enabled:              true,
			Provider:             "Aliyun L4 LB",
			ProtocolsSupported:   []string{"TCP", "UDP"},
			HealthCheckSupported: false,
		},
	}, nil
}
