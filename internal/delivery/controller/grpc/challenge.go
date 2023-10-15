package grpc

import (
	"context"
	"gitlab.com/cyber-ice-box/agent/pkg/controller/grpc/protobuf"
)

type ChallengeService interface {
	Start(ctx context.Context, labId, challengeId string) error
	Stop(ctx context.Context, labId, challengeId string) error
	Reset(ctx context.Context, labId, challengeId string) error
}

func (a *Agent) StartChallenge(ctx context.Context, request *protobuf.ChallengeRequest) (*protobuf.EmptyResponse, error) {

	if err := a.services.Challenges.Start(ctx, request.GetLabId(), request.GetChallengeId()); err != nil {
		return nil, err
	}

	return &protobuf.EmptyResponse{}, nil
}

func (a *Agent) StopChallenge(ctx context.Context, request *protobuf.ChallengeRequest) (*protobuf.EmptyResponse, error) {

	if err := a.services.Challenges.Stop(ctx, request.GetLabId(), request.GetChallengeId()); err != nil {
		return nil, err
	}

	return &protobuf.EmptyResponse{}, nil
}

func (a *Agent) ResetChallenge(ctx context.Context, request *protobuf.ChallengeRequest) (*protobuf.EmptyResponse, error) {

	if err := a.services.Challenges.Reset(ctx, request.GetLabId(), request.GetChallengeId()); err != nil {
		return nil, err
	}

	return &protobuf.EmptyResponse{}, nil
}
