package main

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
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
	"strings"
	"sync"
	"time"
)

const (
	runningStatus = "running"
	none          = "none"
	retries       = 5
	pollInterval  = 30
)

var EnvMutex sync.Mutex

// Driver defines the struct of aliyun driver
type Driver struct {
	driverCapabilities types.Capabilities
	k8sCapabilities    types.K8SCapabilities
}

type state struct {
	// The displayed name of the cluster
	Name string `json:"name,omitempty"`
	//Common fields
	ClusterID                string `json:"cluster_id,omitempty"`
	AccessKeyID              string `json:"accessKeyId,omitempty"`
	AccessKeySecret          string `json:"accessKeySecret,omitempty"`
	DisableRollback          bool   `json:"disable_rollback,omitempty"`
	ClusterType              string `json:"cluster_type,omitempty"`
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
	//not managed Kubernetes fields
	//是否开放公网SSH登录
	SSHFlags                 bool
	MasterInstanceChangeType string
	MasterPeriod             int64
	MasterPeriodUnit         string
	MasterAutoRenew          bool
	MasterAutoRenewPeriod    int64
	MasterInstanceType       string
	MasterSystemDiskCategory string
	MasterSystemDiskSize     int64
	MasterDataDisk           bool
	MasterDataDiskCategory   string
	MasterDataDiskSize       int64
	PublicSlb                bool
	//multi az type options
	MultiAz             bool
	VswitchIDA          string
	VswitchIDB          string
	VswitchIDC          string
	MasterInstanceTypeA string
	MasterInstanceTypeB string
	MasterInstanceTypeC string
	WorkerInstanceTypeA string
	WorkerInstanceTypeB string
	WorkerInstanceTypeC string
	NumOfNodesA         int64
	NumOfNodesB         int64
	NumOfNodesC         int64

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

	//driver.driverCapabilities.AddCapability(types.GetVersionCapability)
	//driver.driverCapabilities.AddCapability(types.SetVersionCapability)
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
		Usage: "the display name of the cluster",
	}
	driverFlag.Options["access-key-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "AcessKeyId",
	}
	driverFlag.Options["access-key-secret"] = &types.Flag{
		Type:  types.StringType,
		Usage: "AccessKeySecret",
	}
	driverFlag.Options["disable-rollback"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "失败是否回滚",
	}
	driverFlag.Options["cluster-type"] = &types.Flag{
		Type:  types.StringType,
		Usage: "集群类型,Kubernetes或ManagedKubernetes",
		Default: &types.Default{
			DefaultString: "ManagedKubernetes",
		},
	}
	driverFlag.Options["timeout-mins"] = &types.Flag{
		Type:  types.IntType,
		Usage: "集群资源栈创建超时时间，以分钟为单位，默认值 60分钟",
	}
	driverFlag.Options["region-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "集群所在地域ID",
	}
	driverFlag.Options["zone-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "所属地域的可用区",
	}
	driverFlag.Options["vpc-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "VPC ID，可空。如果不设置，系统会自动创建VPC，系统创建的VPC网段为192.168.0.0/16。 VpcId 和 vswitchid 只能同时为空或者同时都设置相应的值",
	}
	driverFlag.Options["vswitch-id"] = &types.Flag{
		Type:  types.StringType,
		Usage: "交换机ID，可空。若不设置，系统会自动创建交换机，系统自定创建的交换机网段为 192.168.0.0/16",
	}
	driverFlag.Options["container-cidr"] = &types.Flag{
		Type:  types.StringType,
		Usage: "容器网段，不能和VPC网段冲突。当选择系统自动创建VPC时，默认使用172.16.0.0/16网段",
	}
	driverFlag.Options["service-cidr"] = &types.Flag{
		Type:  types.StringType,
		Usage: "服务网段，不能和VPC网段以及容器网段冲突。当选择系统自动创建VPC时，默认使用172.19.0.0/20网段",
	}
	driverFlag.Options["worker-instance-charge-type"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Worker节点付费类型，可选值为：PrePaid: 预付费; PostPaid: 按量付费",
		Default: &types.Default{
			DefaultString: "PostPaid",
		},
	}
	driverFlag.Options["worker-period-unit"] = &types.Flag{
		Type:  types.StringType,
		Usage: "当指定为PrePaid的时候需要指定周期,Week或Month",
	}
	driverFlag.Options["worker-period"] = &types.Flag{
		Type:  types.IntType,
		Usage: "包年包月时长",
	}
	driverFlag.Options["worker-auto-renew"] = &types.Flag{
		Type:  types.BoolPointerType,
		Usage: "是否开启Worker节点自动续费",
	}
	driverFlag.Options["worker-auto-renew-period"] = &types.Flag{
		Type:  types.IntType,
		Usage: "自动续费周期",
	}
	driverFlag.Options["worker-data-disk"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "是否挂载数据盘",
	}
	driverFlag.Options["worker-data-disk-category"] = &types.Flag{
		Type:  types.StringType,
		Usage: "数据盘类型",
	}

	driverFlag.Options["worker-data-disk-size"] = &types.Flag{
		Type:  types.IntType,
		Usage: "数据盘大小",
	}
	driverFlag.Options["worker-instance-type"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Worker 节点 ECS 规格类型代码",
	}
	driverFlag.Options["worker-system-disk-category"] = &types.Flag{
		Type:  types.StringType,
		Usage: "Worker节点系统盘类型",
	}
	driverFlag.Options["worker-system-disk-size"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Worker节点系统盘大小",
	}
	driverFlag.Options["login-password"] = &types.Flag{
		Type:  types.StringType,
		Usage: "SSH登录密码。密码规则为8 - 30 个字符，且同时包含三项（大、小写字母，数字和特殊符号）。和key_pair 二选一",
	}
	driverFlag.Options["key-pair"] = &types.Flag{
		Type:  types.StringType,
		Usage: "keypair名称。与login_password二选一",
	}
	driverFlag.Options["num-of-nodes"] = &types.Flag{
		Type:  types.IntType,
		Usage: "Worker节点数。范围是[0,300]",
	}
	driverFlag.Options["snat-entry"] = &types.Flag{
		Type:  types.StringType,
		Usage: "是否为网络配置SNAT。如果是自动创建VPC必须设置为true。如果使用已有VPC则根据是否具备出网能力来设置",
		Default: &types.Default{
			DefaultBool: true,
		},
	}
	driverFlag.Options["cloud-monitor-flags"] = &types.Flag{
		Type:  types.BoolType,
		Usage: "是否安装云监控插件",
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
	d.Name = options.GetValueFromDriverOptions(driverOptions, types.StringType, "name").(string)
	d.AccessKeyID = options.GetValueFromDriverOptions(driverOptions, types.StringType, "access-key-id", "accessKeyId").(string)
	d.AccessKeySecret = options.GetValueFromDriverOptions(driverOptions, types.StringType, "access-key-secret", "accessKeySecret").(string)
	d.DisableRollback = options.GetValueFromDriverOptions(driverOptions, types.BoolType, "disable-rollback", "disableRollback").(bool)
	d.ClusterType = options.GetValueFromDriverOptions(driverOptions, types.StringType, "cluster-type", "clusterType").(string)
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
		if cluster.State == runningStatus {
			log.Infof(ctx, "Cluster %v is running", state.Name)
			return nil
		}
		status, err := getClusterLastMessage(svc, state)
		if err != nil {
			return err
		}
		if status != lastMsg {
			log.Infof(ctx, "provisioning cluster %v:......", state.Name, status)
			lastMsg = status
		}
		time.Sleep(time.Second * 5)
	}
}

func getWrapCreateClusterRequest(state *state) (*cs.CreateClusterRequest, error) {
	req := cs.CreateCreateClusterRequest()
	content, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	req.SetScheme("HTTPS")
	req.SetDomain("cs.aliyuncs.com")
	req.SetContentType("application/json;charset=utf-8")
	req.SetContent(content)
	return req, nil
}

func getWrapRemoveClusterRequest(state *state) *cs.DeleteClusterRequest {
	req := cs.CreateDeleteClusterRequest()
	req.ClusterId = state.ClusterID
	req.SetScheme("HTTPS")
	req.SetDomain("cs.aliyuncs.com")
	req.SetContentType("application/json;charset=utf-8")
	return req
}

func getCluster(svc *cs.Client, state *state) (*clusterGetResponse, error) {
	req := cs.CreateDescribeClusterDetailRequest()
	req.ClusterId = state.ClusterID
	req.SetScheme("HTTPS")
	req.SetDomain("cs.aliyuncs.com")
	req.SetContentType("application/json;charset=utf-8")
	resp, err := svc.DescribeClusterDetail(req)
	if err != nil {
		return nil, err
	}
	logrus.Infof("get cluster:\n")
	logrus.Infof(resp.GetHttpContentString())
	cluster := &clusterGetResponse{}
	if err := json.Unmarshal(resp.GetHttpContentBytes(), cluster); err != nil {
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

	req, err := getWrapCreateClusterRequest(state)
	if err != nil {
		return nil, err
	}
	resp, err := svc.CreateCluster(req)
	if err != nil && !strings.Contains(err.Error(), "AlreadyExist") {
		return nil, err
	}
	cluster := &clusterCreateResponse{}
	if err := json.Unmarshal(resp.GetHttpContentBytes(), cluster); err != nil {
		return nil, err
	}

	if err == nil {
		state.ClusterID = cluster.ClusterID
		logrus.Debugf("Cluster %s create is called for region %s and zone %s. Status Code %v", state.ClusterID, state.RegionID, state.ZoneID, resp.GetHttpStatus())
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
	logrus.Info("unimplemented")
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
	info.RootCaCertificate = certs.Ca
	info.ClientCertificate = certs.Cert
	info.ClientKey = certs.Key
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
	req := getWrapRemoveClusterRequest(state)
	resp, err := svc.DeleteCluster(req)
	if err != nil && !strings.Contains(err.Error(), "NotFound") {
		return err
	} else if err == nil {
		logrus.Debugf("Cluster %v delete is called. Status Code %v", state.Name, resp.GetHttpStatus())
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
	logrus.Info("unimplemented")
	return nil
}

func (d *Driver) GetCapabilities(ctx context.Context) (*types.Capabilities, error) {
	return &d.driverCapabilities, nil
}

func (d *Driver) GetK8SCapabilities(ctx context.Context, opts *types.DriverOptions) (*types.K8SCapabilities, error) {
	return &types.K8SCapabilities{
		L4LoadBalancer: &types.L4LoadBalancer{
			Enabled:              false,
			Provider:             "Aliyun L4 LB",
			ProtocolsSupported:   []string{"TCP", "UDP"},
			HealthCheckSupported: false,
		},
	}, nil
}
