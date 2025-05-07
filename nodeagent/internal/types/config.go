package types

type Config struct {
	ClusterID             string
	BootstrapToken        string
	IPAddr                string
	APIAddr               string
	GatewayAddr           string
	StorageDirectory      string
	InitBinary            string
	InitContainerBinary   string
	VMLinux               string
	CloudHypervisorBinary string
	VolumeGroup           string
	ControlPlaneURL       string
}
