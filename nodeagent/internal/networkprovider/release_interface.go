package networkprovider

import (
	"context"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/core/typeutil"
	"github.com/baepo-cloud/baepo-node/nodeagent/internal/types"
	"github.com/vishvananda/netlink"
	"time"
)

func (p *Provider) ReleaseInterface(ctx context.Context, networkInterface *types.NetworkInterface) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	link, err := netlink.LinkByName(networkInterface.Name)
	if err != nil && !isLinkNotFoundError(err) {
		return fmt.Errorf("failed to find interface %s: %w", networkInterface.Name, err)
	}

	if link != nil {
		if err = p.applyTapFirewallRules(ctx, networkInterface, true); err != nil {
			return fmt.Errorf("failed to apply firewall rules to tap interface: %w", err)
		}

		if err = netlink.LinkDel(link); err != nil {
			return fmt.Errorf("failed to delete interface %s: %w", networkInterface.Name, err)
		}
	}

	if networkInterface.ReleasedAt == nil {
		p.allocatedIPs[p.calculateOffsetFromIP(networkInterface.IPAddress)] = ""
		networkInterface.ReleasedAt = typeutil.Ptr(time.Now())
		if err = p.db.WithContext(ctx).Select("ReleasedAt").Save(&networkInterface).Error; err != nil {
			return fmt.Errorf("failed to persist network interface changes in database: %w", err)
		}
	}

	return nil
}

func isLinkNotFoundError(err error) bool {
	var linkNotFoundError netlink.LinkNotFoundError
	return errors.As(err, &linkNotFoundError)
}
