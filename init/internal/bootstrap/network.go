package bootstrap

import (
	"fmt"
	coretypes "github.com/baepo-cloud/baepo-node/core/types"
	"github.com/vishvananda/netlink"
	"net"
)

func SetupNetwork(config coretypes.InitConfig) error {
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("error getting loopback interface: %v", err)
	}

	if err := netlink.LinkSetUp(lo); err != nil {
		return fmt.Errorf("error configuring loopback interface: %v", err)
	}

	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return fmt.Errorf("error getting eth0 interface: %v", err)
	}

	macAddr, err := net.ParseMAC(config.MacAddress)
	if err != nil {
		return fmt.Errorf("failed to parse mac address: %w", err)
	}

	if err = netlink.LinkSetHardwareAddr(eth0, macAddr); err != nil {
		return fmt.Errorf("failed to set mac address: %w", err)
	}

	ipAddr, err := netlink.ParseAddr(config.IPAddress)
	if err != nil {
		return fmt.Errorf("failed to parse ip address: %w", err)
	}

	if err = netlink.AddrAdd(eth0, ipAddr); err != nil {
		return fmt.Errorf("failed to set ip address: %w", err)
	}

	if err = netlink.LinkSetUp(eth0); err != nil {
		return fmt.Errorf("error bringing up eth0: %v", err)
	}

	route := &netlink.Route{
		Gw: net.ParseIP(config.GatewayAddress),
	}
	if err = netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("error adding default route: %v", err)
	}

	return nil
}
