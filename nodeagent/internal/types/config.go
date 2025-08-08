package types

type Config struct {
	Debug            bool
	ClusterID        string
	BootstrapToken   string
	IPAddr           string
	APIAddr          string
	GatewayAddr      string
	StorageDirectory string
	RuntimeBinary    string
	VolumeGroup      string
	ControlPlaneURL  string
}
