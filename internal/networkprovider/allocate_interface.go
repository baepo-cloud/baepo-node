package networkprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
	"github.com/nrednav/cuid2"
	"github.com/vishvananda/netlink"
	"net"
	"strings"
)

func (p *Provider) AllocateInterface(ctx context.Context) (*types.NetworkInterface, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	index := p.findAvailableOffset()
	if index == -1 {
		return nil, fmt.Errorf("no available ip addresses in network %s", p.networkCIDR)
	}

	networkInterface := &types.NetworkInterface{
		ID:             cuid2.Generate(),
		Name:           fmt.Sprintf("tap%d", index),
		GatewayAddress: p.gatewayAddr,
		NetworkCIDR:    typeutil.Ptr(types.GormNetIPNet(*p.networkCIDR)),
	}
	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			Name: networkInterface.Name,
		},
		Mode:  netlink.TUNTAP_MODE_TAP,
		Flags: netlink.TUNTAP_ONE_QUEUE | netlink.TUNTAP_VNET_HDR,
	}
	networkInterface.IPAddress = p.calculateIPFromOffset(index)
	macAddress := fmt.Sprintf("52:54:00:%02x:%02x:%02x",
		(index>>16)&0xFF,
		(index>>8)&0xFF,
		index&0xFF)
	hwAddress, err := net.ParseMAC(macAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mac address: %w", err)
	}
	networkInterface.MacAddress = hwAddress

	if err = p.db.WithContext(ctx).Create(&networkInterface).Error; err != nil {
		return nil, fmt.Errorf("failed to save network interface in database: %w", err)
	}

	if err = netlink.LinkAdd(tap); err != nil {
		return nil, fmt.Errorf("failed to add tap interface: %w", err)
	}

	if err = netlink.LinkSetHardwareAddr(tap, networkInterface.MacAddress); err != nil {
		_ = netlink.LinkDel(tap)
		return nil, fmt.Errorf("failed to set mac address: %w", err)
	}

	if err = netlink.LinkSetUp(tap); err != nil {
		_ = netlink.LinkDel(tap)
		return nil, fmt.Errorf("failed to set tap interface up: %w", err)
	}

	bridge, err := netlink.LinkByName(p.bridgeInterface)
	if err != nil {
		_ = netlink.LinkDel(tap)
		return nil, fmt.Errorf("failed to find bridge %s: %w", p.bridgeInterface, err)
	}

	if err = netlink.LinkSetMaster(tap, bridge); err != nil {
		_ = netlink.LinkDel(tap)
		return nil, fmt.Errorf("failed to set tap interface master: %w", err)
	}

	if err = p.applyTapFirewallRules(ctx, networkInterface, false); err != nil {
		return nil, fmt.Errorf("failed to apply firewall rules to tap interface: %w", err)
	}

	p.allocatedIPs[index] = networkInterface.Name
	return networkInterface, nil
}

func (p *Provider) applyTapFirewallRules(ctx context.Context, networkInterface *types.NetworkInterface, shouldRemove bool) error {
	operation := "-A" // append
	if shouldRemove {
		operation = "-D" // delete
	}

	err := p.runCmd(ctx, "ebtables", operation, "FORWARD",
		"-i", networkInterface.Name,
		"-s", "!", strings.ToLower(networkInterface.MacAddress.String()),
		"-j", "DROP")
	if err != nil {
		return fmt.Errorf("failed to apply mac filtering rule: %w", err)
	}

	err = p.runCmd(ctx, "iptables", operation, "FORWARD",
		"-i", networkInterface.Name,
		"!", "-s", networkInterface.IPAddress.String(),
		"-j", "DROP")
	if err != nil {
		return fmt.Errorf("failed to apply ip filtering rule: %w", err)
	}

	_ = p.runCmd(ctx, "arptables", "-N", "FORWARD")
	err = p.runCmd(ctx, "arptables", operation, "FORWARD",
		"-i", networkInterface.Name,
		"!", "--source-mac", strings.ToLower(networkInterface.MacAddress.String()),
		"-j", "DROP")
	if err != nil {
		return fmt.Errorf("failed to apply arp filtering rule: %w", err)
	}

	return nil
}

func (p *Provider) findAvailableOffset() int {
	for index := 0; index < len(p.allocatedIPs); index++ {
		if p.allocatedIPs[index] == "" {
			return index
		}
	}
	return -1
}
