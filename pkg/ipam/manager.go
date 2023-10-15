package ipam

import (
	"context"
	"errors"
	goipam "github.com/metal-stack/go-ipam"
	"gitlab.com/cyber-ice-box/agent/pkg/postgres"
)

type IPAManager struct {
	ipaManager          goipam.Ipamer
	acquireSingleIPCIDR string
	acquireSubnetCIDR   string
}

const port = "5432"

func NewIPAManager(cfg postgres.Config, defaultAcquireSingleIPCIDR, defaultAcquireSubnetCIDR string) (*IPAManager, error) {
	storage, err := goipam.NewPostgresStorage(
		cfg.Endpoint,
		port,
		cfg.Username,
		cfg.Password,
		cfg.DBName,
		goipam.SSLModeVerifyFull,
	)

	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	ipam := goipam.NewWithStorage(storage)

	if defaultAcquireSingleIPCIDR != "" {
		pr := ipam.PrefixFrom(ctx, defaultAcquireSingleIPCIDR)
		if pr == nil {
			_, err = ipam.NewPrefix(ctx, defaultAcquireSingleIPCIDR)
			if err != nil {
				return nil, err
			}
		}
	}
	if defaultAcquireSubnetCIDR != "" {

		pr := ipam.PrefixFrom(ctx, defaultAcquireSubnetCIDR)
		if pr == nil {
			_, err = ipam.NewPrefix(ctx, defaultAcquireSubnetCIDR)
			if err != nil {
				return nil, err
			}
		}
	}

	return &IPAManager{
		ipaManager:          ipam,
		acquireSingleIPCIDR: defaultAcquireSingleIPCIDR,
		acquireSubnetCIDR:   defaultAcquireSubnetCIDR,
	}, nil
}

func (m *IPAManager) AcquireSingleIP(ctx context.Context, specificIP, specificCIDR *string) (string, error) {
	cidr := m.acquireSingleIPCIDR
	if specificCIDR != nil {
		cidr = *specificCIDR
	}

	if specificIP != nil {
		_, err := m.ipaManager.AcquireSpecificIP(ctx, cidr, *specificIP)
		if err != nil && !errors.Is(err, goipam.ErrAlreadyAllocated) {
			return "", err
		}
		return *specificIP, nil
	}

	ip, err := m.ipaManager.AcquireIP(ctx, cidr)
	if err != nil {
		return "", err
	}
	return ip.IP.String(), nil
}

func (m *IPAManager) ReleaseSingleIP(ctx context.Context, ip string, specificCIDR *string) error {
	cidr := m.acquireSingleIPCIDR
	if specificCIDR != nil {
		cidr = *specificCIDR
	}

	return m.ipaManager.ReleaseIPFromPrefix(ctx, cidr, ip)
}

func (m *IPAManager) AcquireChildSubnet(ctx context.Context, blockSize uint32, specificCIDR *string) (string, error) {
	cidr := m.acquireSubnetCIDR
	if specificCIDR != nil {
		cidr = *specificCIDR
	}

	prefix, err := m.ipaManager.AcquireChildPrefix(ctx, cidr, uint8(blockSize))
	if err != nil {
		return "", err
	}
	return prefix.Cidr, nil
}

func (m *IPAManager) ReleaseChildSubnet(ctx context.Context, childSubnet string) error {
	_, err := m.ipaManager.DeletePrefix(ctx, childSubnet)

	return err
}
