package types

import (
	"context"
	"database/sql/driver"
	"errors"
	"net"
	"time"
)

type (
	GormNetIPNet net.IPNet

	NetworkInterface struct {
		ID             string `gorm:"primaryKey"`
		Name           string
		IPAddress      net.IP           `gorm:"type:text"`
		MacAddress     net.HardwareAddr `gorm:"type:text"`
		GatewayAddress net.IP           `gorm:"type:text"`
		NetworkCIDR    *GormNetIPNet    `gorm:"column:network_cidr"`
		AllocatedAt    *time.Time
		ReleasedAt     *time.Time
		CreatedAt      time.Time
	}

	NetworkProvider interface {
		GetInterface(name string) (*NetworkInterface, error)

		AllocateInterface(ctx context.Context) (*NetworkInterface, error)

		SetupInterface(ctx context.Context, networkInterface *NetworkInterface) error

		ReleaseInterface(ctx context.Context, networkInterface *NetworkInterface) error
	}
)

var ErrNetworkInterfaceNotFound = errors.New("network interface not found")

func (*GormNetIPNet) GormDataType() string {
	return "text"
}

func (v *GormNetIPNet) Scan(value interface{}) error {
	_, ipNet, err := net.ParseCIDR(value.(string))
	if err != nil {
		return err
	}

	*v = GormNetIPNet(*ipNet)
	return nil
}

func (v *GormNetIPNet) Value() (driver.Value, error) {
	ipNet := net.IPNet(*v)
	return ipNet.String(), nil
}
