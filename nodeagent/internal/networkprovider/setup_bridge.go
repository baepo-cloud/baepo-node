package networkprovider

import (
	"context"
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
	"os"
	"strings"
)

var LocalNetworks = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

func (p *Provider) SetupBridge(externalInterface string) (netlink.Link, error) {
	bridge, err := netlink.LinkByName(p.bridgeInterface)
	if err != nil {
		attrs := netlink.NewLinkAttrs()
		attrs.Name = p.bridgeInterface
		bridge = &netlink.Bridge{LinkAttrs: attrs}
		if err = netlink.LinkAdd(bridge); err != nil {
			return nil, fmt.Errorf("failed to create bridge: %w", err)
		}
	}

	p.networkAddr = p.networkCIDR.IP
	p.gatewayAddr = p.calculateIPFromOffset(1)
	addr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   p.gatewayAddr,
			Mask: p.networkCIDR.Mask,
		},
	}

	if err = netlink.AddrAdd(bridge, addr); err != nil && !strings.Contains(err.Error(), "file exists") {
		return nil, fmt.Errorf("failed to add address to bridge: %w", err)
	}

	if err = netlink.LinkSetUp(bridge); err != nil {
		return nil, fmt.Errorf("failed to bring up bridge: %w", err)
	}

	// enable IP forwarding in the kernel
	if err = os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644); err != nil {
		return nil, fmt.Errorf("failed to setup IP forwarding: %w", err)
	}

	if err = p.applyBridgeFirewallRules(externalInterface); err != nil {
		return nil, fmt.Errorf("failed to setup firewall rules: %w", err)
	}

	return bridge, nil
}

func (p *Provider) applyBridgeFirewallRules(externalInterface string) error {
	// NAT for external access
	err := p.upsertIptablesRule("nat", "POSTROUTING",
		"-s", p.networkCIDR.String(),
		"-o", externalInterface,
		"-j", "MASQUERADE")
	if err != nil {
		return fmt.Errorf("failed to setup NAT: %w", err)
	}

	// Allow forwarding for VM traffic going to external networks
	err = p.upsertIptablesRule("filter", "FORWARD",
		"-i", p.bridgeInterface,
		"-o", externalInterface,
		"-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("failed to allow VM outbound traffic: %w", err)
	}

	// Allow established connections
	err = p.upsertIptablesRule("filter", "FORWARD",
		"-i", externalInterface,
		"-o", p.bridgeInterface,
		"-m", "state",
		"--state", "RELATED,ESTABLISHED",
		"-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("failed to allow established connections: %w", err)
	}

	// Block VM access to local networks
	for _, network := range LocalNetworks {
		err = p.upsertIptablesRule("filter", "FORWARD",
			"-i", p.bridgeInterface,
			"-o", externalInterface,
			"-d", network,
			"-j", "DROP")
		if err != nil {
			return fmt.Errorf("failed to block VM access to local network %s: %w", network, err)
		}
	}

	return nil
}

func (p *Provider) upsertIptablesRule(table, chain string, ruleSpec ...string) error {
	checkArgs := []string{"-t", table, "-C", chain}
	checkArgs = append(checkArgs, ruleSpec...)
	if err := p.runCmd(context.Background(), "iptables", checkArgs...); err == nil {
		return nil
	}

	addArgs := []string{"-t", table, "-A", chain}
	addArgs = append(addArgs, ruleSpec...)
	return p.runCmd(context.Background(), "iptables", addArgs...)
}
