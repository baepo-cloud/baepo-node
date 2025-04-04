package types

import "net"

type (
	NetworkInterface struct {
		Name       string
		IPAddress  net.IP
		MacAddress net.HardwareAddr
	}

	NetworkProvider interface {
		AllocateInterface() (*NetworkInterface, error)

		ReleaseInterface(name string) error
	}
)
