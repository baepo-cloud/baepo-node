package networkprovider

import (
	"context"
	"fmt"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/vishvananda/netlink"
	"strings"
)

func (p *Provider) SetupInterface(ctx context.Context, networkInterface *types.NetworkInterface) error {
	if link, err := netlink.LinkByName(networkInterface.Name); err == nil {
		if err = netlink.LinkDel(link); err != nil {
			return fmt.Errorf("failed to delete existing tap interface %s: %w", networkInterface.Name, err)
		}
	}

	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			Name: networkInterface.Name,
		},
		Mode:  netlink.TUNTAP_MODE_TAP,
		Flags: netlink.TUNTAP_ONE_QUEUE | netlink.TUNTAP_VNET_HDR,
	}
	if err := netlink.LinkAdd(tap); err != nil {
		return fmt.Errorf("failed to add tap interface: %w", err)
	}

	if err := netlink.LinkSetHardwareAddr(tap, networkInterface.MacAddress); err != nil {
		_ = netlink.LinkDel(tap)
		return fmt.Errorf("failed to set mac address: %w", err)
	}

	if err := netlink.LinkSetUp(tap); err != nil {
		_ = netlink.LinkDel(tap)
		return fmt.Errorf("failed to set tap interface up: %w", err)
	}

	bridge, err := netlink.LinkByName(p.bridgeInterface)
	if err != nil {
		_ = netlink.LinkDel(tap)
		return fmt.Errorf("failed to find bridge %s: %w", p.bridgeInterface, err)
	}

	if err = netlink.LinkSetMaster(tap, bridge); err != nil {
		_ = netlink.LinkDel(tap)
		return fmt.Errorf("failed to set tap interface master: %w", err)
	}

	if err = p.applyTapFirewallRules(ctx, networkInterface, false); err != nil {
		return fmt.Errorf("failed to apply firewall rules to tap interface: %w", err)
	}

	return nil
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
