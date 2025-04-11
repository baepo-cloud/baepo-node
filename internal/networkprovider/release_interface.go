package networkprovider

import (
	"context"
	"fmt"
	"github.com/vishvananda/netlink"
	"log/slog"
)

func (p *Provider) ReleaseInterface(name string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	index := -1
	for i, tapName := range p.allocatedIPs {
		if tapName == name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("interface %s not found", name)
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to find interface %s: %w", name, err)
	}

	if err = netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete interface %s: %w", name, err)
	}

	if err = p.runCmd(context.Background(), "ebtables", "-D", "FORWARD", "-i", name, "-j", "DROP"); err != nil {
		slog.Error("failed to remove mac filtering rule", slog.Any("error", err))
	}
	if err = p.runCmd(context.Background(), "iptables", "-D", "FORWARD", "-i", name, "-j", "DROP"); err != nil {
		slog.Error("failed to remove ip filtering rule", slog.Any("error", err))
	}
	if err = p.runCmd(context.Background(), "arptables", "-D", "FORWARD", "-i", name, "-j", "DROP"); err != nil {
		slog.Error("failed to remove arp filtering rule", slog.Any("error", err))
	}

	p.allocatedIPs[index] = ""
	return nil
}
