package challenge

import (
	"context"
	"fmt"
	"gitlab.com/cyber-ice-box/agent/internal/model/delivery"
	"gitlab.com/cyber-ice-box/agent/internal/model/service"
)

const challengeIdLabel = "challengeId"

type (
	k8s interface {
		ApplyDeployment(ctx context.Context, config delivery.ApplyDeploymentConfig) error
		GetDeploymentsInNamespaceBySelector(ctx context.Context, namespace string, selector ...string) ([]delivery.DeploymentStatus, error)
		ResetDeployment(ctx context.Context, name, namespace string) error
		ScaleDeployment(ctx context.Context, name, namespace string, scale int32) error
		DeleteDeployment(ctx context.Context, name, namespace string) error
	}

	ipaManager interface {
		AcquireSingleIP(ctx context.Context, specificIP, specificCIDR *string) (string, error)
		ReleaseSingleIP(ctx context.Context, ip string, specificCIDR *string) error
	}

	Service struct {
		k8s        k8s
		ipaManager ipaManager
	}
)

func New(k8s k8s, ipaManager ipaManager) *Service {
	return &Service{k8s: k8s, ipaManager: ipaManager}
}

func (ch *Service) Create(ctx context.Context, labId, labCIDR string, challengeConfig service.ChallengeConfig) (records []service.RecordConfig, errs error) {
	for i, inst := range challengeConfig.Instances {
		ip, err := ch.ipaManager.AcquireSingleIP(ctx, nil, &labCIDR)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		if err = ch.k8s.ApplyDeployment(ctx, delivery.ApplyDeploymentConfig{
			Name:      fmt.Sprintf("%s-%d", challengeConfig.Id, i),
			Namespace: labId,
			Label: delivery.Label{
				Key:   challengeIdLabel,
				Value: challengeConfig.Id,
			},
			Image:     inst.Image,
			Resources: inst.Resources,
			Envs:      inst.Envs,
			Ip:        ip,
		}); err != nil {
			errs = multierror.Append(errs, err)
			if err = ch.ipaManager.ReleaseSingleIP(ctx, ip, &labCIDR); err != nil {
				errs = multierror.Append(errs, err)
			}
			continue
		}

		for _, r := range inst.Records {
			if r.Type == "A" {
				r.Data = ip
			}
			records = append(records, r)
		}
	}
	return
}

func (ch *Service) Start(ctx context.Context, labId, challengeId string) (errs error) {
	dps, err := ch.k8s.GetDeploymentsInNamespaceBySelector(ctx, labId, fmt.Sprintf("%s:%s", challengeId, challengeId))
	if err != nil {
		return err
	}

	for _, dp := range dps {
		if err = ch.k8s.ScaleDeployment(ctx, dp.Name, labId, 1); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return
}

func (ch *Service) Stop(ctx context.Context, labId, challengeId string) (errs error) {
	dps, err := ch.k8s.GetDeploymentsInNamespaceBySelector(ctx, labId, fmt.Sprintf("%s:%s", challengeId, challengeId))
	if err != nil {
		return err
	}

	for _, dp := range dps {
		if err = ch.k8s.ScaleDeployment(ctx, dp.Name, labId, 0); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return
}

func (ch *Service) Reset(ctx context.Context, labId, challengeId string) (errs error) {
	dps, err := ch.k8s.GetDeploymentsInNamespaceBySelector(ctx, labId, fmt.Sprintf("%s:%s", challengeId, challengeId))
	if err != nil {
		return err
	}

	for _, dp := range dps {
		if err = ch.k8s.ResetDeployment(ctx, dp.Name, labId); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return
}

func (ch *Service) Delete(ctx context.Context, labId, labCIDR, challengeId string) (errs error) {
	dps, err := ch.k8s.GetDeploymentsInNamespaceBySelector(ctx, labId, fmt.Sprintf("%s:%s", challengeId, challengeId))
	if err != nil {
		return err
	}

	for _, dp := range dps {
		if err = ch.k8s.DeleteDeployment(ctx, dp.Name, labId); err != nil {
			errs = multierror.Append(errs, err)
		}

		if err = ch.ipaManager.ReleaseSingleIP(ctx, dp.IP, &labCIDR); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return
}
