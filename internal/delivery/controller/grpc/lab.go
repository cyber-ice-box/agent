package grpc

import (
	"context"
	"github.com/rs/zerolog/log"
	"gitlab.com/cyber-ice-box/agent/internal/model/service"
	"gitlab.com/cyber-ice-box/agent/pkg/controller/grpc/protobuf"
)

type LabService interface {
	Create(ctx context.Context, labId string, cidrBlockSize uint32) (labCIDR string, err error)
	Delete(ctx context.Context, labId string) error
	AddChallenges(ctx context.Context, labIds []string, challengeConfigs []service.ChallengeConfig) error
	DeleteChallenges(ctx context.Context, labIds, challengeIds []string) error
}

func (a *Agent) CreateLab(ctx context.Context, request *protobuf.CreateLabRequest) (*protobuf.CreateLabResponse, error) {
	cidr, err := a.services.Labs.Create(ctx, request.Id, request.CidrBlockSize)
	if err != nil {
		return nil, err
	}

	return &protobuf.CreateLabResponse{
		Cidr: cidr,
	}, nil
}

func (a *Agent) DeleteLab(ctx context.Context, request *protobuf.DeleteLabRequest) (*protobuf.EmptyResponse, error) {
	err := a.services.Labs.Delete(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	return &protobuf.EmptyResponse{}, nil
}

func (a *Agent) AddChallenges(ctx context.Context, request *protobuf.AddChallengesRequest) (*protobuf.EmptyResponse, error) {
	challengesConfigs := make([]service.ChallengeConfig, 0)

	for _, chConfig := range request.GetChallenges() {
		instances := make([]service.InstanceConfig, 0)

		for _, inst := range chConfig.GetInstances() {
			envs := make([]service.EnvConfig, 0)
			for _, env := range inst.GetEnvs() {
				envs = append(envs, service.EnvConfig{
					Name:  env.GetName(),
					Value: env.GetValue(),
				})
			}

			records := make([]service.RecordConfig, 0)
			for _, record := range inst.GetRecords() {
				records = append(records, service.RecordConfig{
					Type: record.GetType(),
					Name: record.GetName(),
					Data: record.GetData(),
				})
			}

			instances = append(instances, service.InstanceConfig{
				Image: inst.GetImage(),
				Resources: service.ResourcesConfig{
					Requests: service.ResourceConfig{
						Memory: inst.Resources.GetMemory(),
						CPU:    inst.Resources.GetCpu(),
					},
					Limit: service.ResourceConfig{},
				},
				Envs:    envs,
				Records: records,
			})
		}

		challengesConfigs = append(challengesConfigs, service.ChallengeConfig{Id: chConfig.GetId(), Instances: instances})
	}
	log.Info().Msg("add cha agent")
	if err := a.services.Labs.AddChallenges(ctx, request.GetLabIds(), challengesConfigs); err != nil {
		return nil, err
	}

	return &protobuf.EmptyResponse{}, nil
}

func (a *Agent) DeleteChallenges(ctx context.Context, request *protobuf.DeleteChallengesRequest) (*protobuf.EmptyResponse, error) {
	err := a.services.Labs.DeleteChallenges(ctx, request.GetLabIds(), request.GetChallengeIds())
	if err != nil {
		return nil, err
	}

	return &protobuf.EmptyResponse{}, nil
}
