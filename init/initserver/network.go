package initserver

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
)

func (s *Server) setupNetwork() error {
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

	macAddr, err := net.ParseMAC(s.config.MacAddress)
	if err != nil {
		return fmt.Errorf("failed to parse mac address: %w", err)
	}

	if err = netlink.LinkSetHardwareAddr(eth0, macAddr); err != nil {
		return fmt.Errorf("failed to set mac address: %w", err)
	}

	ipAddr, err := netlink.ParseAddr(s.config.IPAddress)
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
		Gw: net.ParseIP(s.config.GatewayAddress),
	}
	if err = netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("error adding default route: %v", err)
	}

	return nil
}
