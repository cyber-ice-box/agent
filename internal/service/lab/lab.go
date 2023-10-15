package lab

import (
	"context"
	"errors"
	"gitlab.com/cyber-ice-box/agent/internal/model/service"
)

type (
	k8s interface {
		ApplyNetwork(ctx context.Context, name, cidr string, blockSize int) error
		GetNetworkCIDR(ctx context.Context, name string) (string, error)
		DeleteNetwork(ctx context.Context, name string) error

		ApplyNamespace(ctx context.Context, name string, ipPoolName *string) error
		NamespaceExists(ctx context.Context, name string) (bool, error)
		DeleteNamespace(ctx context.Context, name string) error
	}

	ipaManager interface {
		AcquireSingleIP(ctx context.Context, specificIP, specificCIDR *string) (string, error)
		AcquireChildSubnet(ctx context.Context, blockSize uint32, specificCIDR *string) (string, error)
		ReleaseChildSubnet(ctx context.Context, childSubnet string) error
	}

	dnsService interface {
		Create(ctx context.Context, labId, ip string) error
		RefreshRecords(ctx context.Context, labId string, records []service.RecordConfig) error
	}

	challengeService interface {
		Create(ctx context.Context, labId, labCIDR string, challengeConfig service.ChallengeConfig) (records []service.RecordConfig, errs error)
		Delete(ctx context.Context, labId, labCIDR, challengeId string) error
	}

	Service struct {
		k8s              k8s
		ipaManager       ipaManager
		dnsService       dnsService
		challengeService challengeService
	}
)

func New(k8s k8s, ipaManager ipaManager, challengeService challengeService, dnsService dnsService) *Service {
	return &Service{k8s: k8s, ipaManager: ipaManager, dnsService: dnsService, challengeService: challengeService}
}

func (l *Service) Create(ctx context.Context, labId string, cidrBlockSize uint32) (labCIDR string, err error) {
	exists, err := l.k8s.NamespaceExists(ctx, labId)
	if err != nil {
		return "", err
	}
	if exists {
		return "", errors.New("lab already exists")
	}

	labCIDR, err = l.ipaManager.AcquireChildSubnet(ctx, cidrBlockSize, nil)
	if err != nil {
		return "", err
	}

	if err = l.k8s.ApplyNetwork(ctx, labId, labCIDR, int(cidrBlockSize)); err != nil {
		if err = l.ipaManager.ReleaseChildSubnet(ctx, labCIDR); err != nil {
			return "", err
		}
		return "", err
	}

	if err = l.k8s.ApplyNamespace(ctx, labId, &labId); err != nil {
		if err = l.ipaManager.ReleaseChildSubnet(ctx, labCIDR); err != nil {
			return "", err
		}
		return "", err
	}

	singleIP, err := l.ipaManager.AcquireSingleIP(ctx, nil, &labCIDR)
	if err != nil {
		return "", err
	}

	if err = l.dnsService.Create(ctx, labId, singleIP); err != nil {
		return "", err
	}

	return
}

func (l *Service) Delete(ctx context.Context, labId string) error {

	if err := l.k8s.DeleteNamespace(ctx, labId); err != nil {
		return err
	}

	cidr, err := l.k8s.GetNetworkCIDR(ctx, labId)
	if err != nil {
		return err
	}

	if err = l.k8s.DeleteNetwork(ctx, labId); err != nil {
		return err
	}

	if err = l.ipaManager.ReleaseChildSubnet(ctx, cidr); err != nil {
		return err
	}
	return nil
}

func (l *Service) AddChallenges(ctx context.Context, labIds []string, challengeConfigs []service.ChallengeConfig) (errs error) {

	for _, labId := range labIds {
		labCIDR, err := l.k8s.GetNetworkCIDR(ctx, labId)

		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		for _, chConfig := range challengeConfigs {
			records, err := l.challengeService.Create(ctx, labId, labCIDR, chConfig)
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}

			if err = l.dnsService.RefreshRecords(ctx, labId, records); err != nil {
				errs = multierror.Append(errs, err)
			}
		}
	}
	return
}

func (l *Service) DeleteChallenges(ctx context.Context, labIds, challengeIds []string) error {
	var errs error
	for _, labId := range labIds {
		labCIDR, err := l.k8s.GetNetworkCIDR(ctx, labId)
		if err != nil {
			return err
		}
		for _, chId := range challengeIds {
			if err = l.challengeService.Delete(ctx, labId, labCIDR, chId); err != nil {
				errs = multierror.Append(errs, err)
			}
		}
	}
	return errs
}
