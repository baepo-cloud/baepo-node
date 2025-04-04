package initd

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"os"
	"strings"
)

var (
	dnsServers   = []string{"1.1.1.1", "1.0.0.1", "2606:4700:4700::1111", "2606:4700:4700::1001"}
	defaultHosts = []string{
		"127.0.0.1 localhost.localdomain localhost",
		"::1 ip6-localhost ip6-loopback",
		"fe00::0 ip6-localnet",
		"ff00::0 ip6-mcastprefix",
		"ff02::1 ip6-allnodes",
		"ff02::2 ip6-allrouters",
		"ff02::3 ip6-allhosts",
	}
)

func (d *initd) SetupNetwork() error {
	resolvEntries := make([]string, len(dnsServers))
	for index, server := range dnsServers {
		resolvEntries[index] = fmt.Sprintf("nameserver %v", server)
	}
	if err := os.WriteFile("/etc/resolv.conf", []byte(strings.Join(resolvEntries, "\n")+"\n"), 0x0755); err != nil {
		return fmt.Errorf("failed to write /etc/resolv.conf: %w", err)
	}

	if err := os.WriteFile("/etc/hosts", []byte(strings.Join(defaultHosts, "\n")+"\n"), 0x0755); err != nil {
		return fmt.Errorf("failed to write /etc/hosts: %w", err)
	}

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

	if err = netlink.LinkSetHardwareAddr(eth0, d.config.MacAddress); err != nil {
		return fmt.Errorf("failed to set mac address: %w", err)
	}

	if err = netlink.AddrAdd(eth0, d.config.IPAddress); err != nil {
		return fmt.Errorf("failed to set ip address: %w", err)
	}

	if err = netlink.LinkSetUp(eth0); err != nil {
		return fmt.Errorf("error bringing up eth0: %v", err)
	}

	// Add default route
	route := &netlink.Route{
		Dst: nil, // default route
		Gw:  d.config.GatewayAddress,
	}

	if err = netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("error adding default route: %v", err)
	}

	return nil
}
