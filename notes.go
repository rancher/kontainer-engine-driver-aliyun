package main

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
	DisableRollback bool   `json:"disableRollback,omitempty"`
	Name            string `json:"name,omitempty"`
	ClusterType     string `json:"clusterYype,omitempty"`
	TimeoutMins     int    `json:"timeoutMins,omitempty"`
	//https://www.alibabacloud.com/help/zh/doc-detail/40654.htm
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
	//https://help.aliyun.com/document_detail/25378.html
	WorkerInstanceType string `json:"workerInstanceType,omitempty"`
	//https://www.alibabacloud.com/help/zh/doc-detail/25691.htm
	WorkerSystemDiskCategory string `json:"workerSystemDiskCategory,omitempty"`
	WorkerSystemDiskSize     int    `json:"workerSystemDiskSize,omitempty"`
	WorkerDataDisk           bool   `json:"workerDataDisk,omitempty"`
	WorkerDataDiskCategory   string `json:"workerDataDiskCategory,omitempty"`
	WorkerDataDiskSize       int    `json:"workerDataDiskSize,omitempty"`
	NumOfNodes               int    `json:"numOfNodes,omitempty"`
	SnatEntry                bool   `json:"snatEntry,omitempty"`

	//not managed Kubernetes fields
	//是否开放公网SSH登录
	SSHFlags                 bool   `json:"sshFlags,omitempty"`
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
	VswitchIDA          string `json:"vswitchIdA,omitempty"`
	VswitchIDB          string `json:"vswitchIdB,omitempty"`
	VswitchIDC          string `json:"vswitchIdC,omitempty"`
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

const example = `
{
  "instance_type": "ecs.n1.medium",
  "vpc_id": "vpc-2zeuhrr3seyrcb1bwz52u",
  "vswitch_id": "vsw-2zeqzoym2ak33yo9yb8n5",
  "vswitch_cidr": "",
  "data_disk_size": 0,
  "data_disk_category": "cloud",
  "security_group_id": "sg-2zehao0t223ml13ebybn",
  "tags": "",
  "zone_id": "cn-beijing-a",
  "-": "PayByTraffic",
  "name": "lawrtest1",
  "cluster_id": "c14a3a52712894dd79b80a73b60c0da67",
  "size": 2,
  "region_id": "cn-beijing",
  "network_mode": "vpc",
  "subnet_cidr": "172.16.0.0/16",
  "state": "running",
  "master_url": "",
  "external_loadbalancer_id": "lb-2zex4dgrptn7wwogafpab",
  "created": "2018-10-30T17:43:01+08:00",
  "updated": "2018-10-30T17:48:46+08:00",
  "port": 0,
  "node_status": "",
  "cluster_healthy": "",
  "docker_version": "17.06.2-ce-3",
  "cluster_type": "ManagedKubernetes",
  "swarm_mode": false,
  "init_version": "1.11.2",
  "current_version": "1.11.2",
  "meta_data": "{\"DockerVersion\":\"17.06.2-ce-3\",\"EtcdVersion\":\"v3.3.8\",\"KubernetesVersion\":\"1.11.2\",\"MultiAZ\":false,\"SubClass\":\"default\"}",
  "gw_bridge": "",
  "upgrade_components": {
    "Kubernetes": {
      "component_name": "Kubernetes",
      "version": "",
      "next_version": "",
      "changed": false,
      "can_upgrade": false,
      "force": false,
      "policy": "",
      "ExtraVars": null,
      "ready_to_upgrade": "",
      "message": ""
    }
  },
  "private_zone": false,
  "capabilities": null,
  "enabled_migration": false,
  "need_update_agent": false,
  "outputs": [
    {
      "Description": "Log Info Output",
      "OutputKey": "LastKnownError",
      "OutputValue": null
    },
    {
      "Description": "Ids of worker node",
      "OutputKey": "NodeInstanceIDs",
      "OutputValue": [
        "i-2zecy1qgs64b6nh0u5vn",
        "i-2zecy1qgs64b6nh0u5vo"
      ]
    }
  ],
  "parameters": {
    "ALIYUN::AccountId": "1617067603288521",
    "ALIYUN::NoValue": "None",
    "ALIYUN::Region": "cn-beijing",
    "ALIYUN::StackId": "2b0af222-455f-4091-a10b-bb919b92a523",
    "ALIYUN::StackName": "k8s-for-cs-c14a3a52712894dd79b80a73b60c0da67",
    "CloudMonitorFlags": "False",
    "CloudMonitorVersion": "1.2.21",
    "ContainerCIDR": "172.16.0.0/16",
    "DockerVersion": "17.06.2-ce-3",
    "Eip": "True",
    "EipAddress": "",
    "EtcdVersion": "v3.3.8",
    "ImageId": "centos_7_04_64_20G_alibase_201701015.vhd",
    "K8sWorkerPolicyDocument": "{\"Version\": \"1\", \"Statement\": [{\"Action\": [\"ecs:AttachDisk\", \"ecs:DetachDisk\", \"ecs:DescribeDisks\", \"ecs:CreateDisk\", \"ecs:CreateSnapshot\", \"ecs:DeleteDisk\", \"ecs:CreateNetworkInterface\", \"ecs:DescribeNetworkInterfaces\", \"ecs:AttachNetworkInterface\", \"ecs:DetachNetworkInterface\", \"ecs:DeleteNetworkInterface\", \"ecs:DescribeInstanceAttribute\"], \"Resource\": [\"*\"], \"Effect\": \"Allow\"}, {\"Action\": [\"nas:*\"], \"Resource\": [\"*\"], \"Effect\": \"Allow\"}, {\"Action\": [\"oss:*\"], \"Resource\": [\"*\"], \"Effect\": \"Allow\"}, {\"Action\": [\"log:*\"], \"Resource\": [\"*\"], \"Effect\": \"Allow\"}, {\"Action\": [\"cms:*\"], \"Resource\": [\"*\"], \"Effect\": \"Allow\"}, {\"Action\": [\"cr:Get*\", \"cr:List*\", \"cr:PullRepository\"], \"Resource\": [\"*\"], \"Effect\": \"Allow\"}]}",
    "KeyPair": "",
    "KubernetesVersion": "1.11.2",
    "LoginPassword": "******",
    "MasterSLBPrivateIP": "192.168.1.217",
    "NatGateway": "True",
    "NatGatewayId": "",
    "NumOfNodes": "2",
    "SNatEntry": "True",
    "ScaleOutToken": "f22b79.4cb1a3886db68cb2",
    "SecurityGroupId": "sg-2zehao0t223ml13ebybn",
    "ServiceCIDR": "172.19.0.0/20",
    "SnatTableId": "",
    "VSwitchId": "vsw-2zeqzoym2ak33yo9yb8n5",
    "VpcId": "vpc-2zeuhrr3seyrcb1bwz52u",
    "WorkerAutoRenew": "False",
    "WorkerAutoRenewPeriod": "1",
    "WorkerDataDisk": "False",
    "WorkerDataDiskCategory": "cloud_ssd",
    "WorkerDataDiskDevice": "/dev/xvdb",
    "WorkerDataDiskSize": "40",
    "WorkerInstanceChargeType": "PostPaid",
    "WorkerInstanceType": "ecs.n1.medium",
    "WorkerPeriod": "3",
    "WorkerPeriodUnit": "Month",
    "WorkerSystemDiskCategory": "cloud_ssd",
    "WorkerSystemDiskSize": "100",
    "ZoneId": "cn-beijing-a"
  }
}
`
