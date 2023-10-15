package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/cyber-ice-box/agent/internal/config"
	"gitlab.com/cyber-ice-box/agent/pkg/controller/grpc/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"os"
)

type Services struct {
	Labs       LabService
	Challenges ChallengeService
}

type Agent struct {
	auth     Authenticator
	config   *config.GRPCConfig
	services Services
	protobuf.UnimplementedAgentServer
}

func getCreds(conf *config.TLSConfig) (credentials.TransportCredentials, error) {
	log.Printf("Preparing credentials for RPC")

	certificate, err := tls.LoadX509KeyPair(conf.CertFile, conf.CertKey)
	if err != nil {
		return nil, fmt.Errorf("could not load server key pair: %s", err)
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(conf.CAFile)
	if err != nil {
		return nil, fmt.Errorf("could not read ca certificate: %s", err)
	}
	// CA file for let's encrypt is located under domain conf as `chain.pem`
	// pass chain.pem location
	// Append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.New("failed to append client certs")
	}

	// Create the TLS credentials
	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	})
	return creds, nil
}

func secureConn(conf *config.TLSConfig) ([]grpc.ServerOption, error) {
	if conf.Enabled {
		log.Info().Msgf("Conf cert-file: %s, cert-key: %s ca: %s", conf.CertFile, conf.CertKey, conf.CAFile)
		creds, err := getCreds(conf)

		if err != nil {
			return []grpc.ServerOption{}, errors.New("Error on retrieving certificates: " + err.Error())
		}
		log.Printf("Server is running in secure mode!")
		return []grpc.ServerOption{grpc.Creds(creds)}, nil
	}
	return []grpc.ServerOption{}, nil
}

func (a *Agent) addAuth(opts ...grpc.ServerOption) *grpc.Server {
	streamInterceptor := func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := a.auth.AuthenticateContext(stream.Context()); err != nil {
			return err
		}
		return handler(srv, stream)
	}

	unaryInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := a.auth.AuthenticateContext(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}

	opts = append([]grpc.ServerOption{
		grpc.StreamInterceptor(streamInterceptor),
		grpc.UnaryInterceptor(unaryInterceptor),
	}, opts...)
	return grpc.NewServer(opts...)

}

func New(conf *config.GRPCConfig, services Services) (*grpc.Server, error) {
	gRPCServer := &Agent{
		auth:     NewAuthenticator(conf.Auth.SignKey, conf.Auth.AuthKey),
		config:   conf,
		services: services,
	}

	opts, err := secureConn(&conf.TLS)
	if err != nil {
		return nil, err
	}

	gRPCEndpoint := gRPCServer.addAuth(opts...)

	reflection.Register(gRPCEndpoint)
	protobuf.RegisterAgentServer(gRPCEndpoint, gRPCServer)

	return gRPCEndpoint, nil
}
