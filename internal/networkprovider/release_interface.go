package networkprovider

import (
	"context"
	"errors"
	"fmt"
	"github.com/baepo-cloud/baepo-node/internal/types"
	"github.com/baepo-cloud/baepo-node/internal/typeutil"
	"github.com/vishvananda/netlink"
	"gorm.io/gorm"
	"time"
)

func (p *Provider) ReleaseInterface(ctx context.Context, name string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	var networkInterface types.NetworkInterface
	err := p.db.WithContext(ctx).First(&networkInterface, "name = ? AND deleted_at IS NULL", name).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to retrieve network interface: %w", err)
	}

	link, err := netlink.LinkByName(networkInterface.Name)
	if err != nil {
		return fmt.Errorf("failed to find interface %s: %w", networkInterface.Name, err)
	}

	networkInterface.DeletedAt = typeutil.Ptr(time.Now())
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err = tx.WithContext(ctx).Save(&networkInterface).Error; err != nil {
			return fmt.Errorf("failed to save network interface in databaae: %w", err)
		}

		if err = p.applyTapFirewallRules(ctx, &networkInterface, true); err != nil {
			return fmt.Errorf("failed to apply firewall rules to tap interface: %w", err)
		}

		if err = netlink.LinkDel(link); err != nil {
			return fmt.Errorf("failed to delete interface %s: %w", networkInterface.Name, err)
		}

		p.allocatedIPs[p.calculateOffsetFromIP(networkInterface.IPAddress)] = ""
		return nil
	})
}
