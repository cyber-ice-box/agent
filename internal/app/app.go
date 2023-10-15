package app

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/cyber-ice-box/agent/internal/config"
	"gitlab.com/cyber-ice-box/agent/internal/delivery/controller/grpc"
	"gitlab.com/cyber-ice-box/agent/internal/delivery/infrastructure/k8s"
	"gitlab.com/cyber-ice-box/agent/internal/service/challenge"
	"gitlab.com/cyber-ice-box/agent/internal/service/dns"
	"gitlab.com/cyber-ice-box/agent/internal/service/lab"
	"gitlab.com/cyber-ice-box/agent/pkg/ipam"
	"gitlab.com/cyber-ice-box/agent/pkg/postgres"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func Run() {
	cfg, valid := config.GetConfig(nil)
	if !valid {
		os.Exit(1)
	}

	ipaManager, err := ipam.NewIPAManager(postgres.Config(cfg.PostgresDB), "", cfg.LabsSubnet)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize IPAManager")
	}
	clusterInfrastructure := k8s.New()

	lis, err := net.Listen("tcp", fmt.Sprintf("%s", cfg.GRPC.Endpoint))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen")
	}

	challengeService := challenge.New(clusterInfrastructure, ipaManager)
	labService := lab.New(clusterInfrastructure, ipaManager, challengeService, dns.New(clusterInfrastructure))

	grpcServer, err := grpc.New(&cfg.GRPC, grpc.Services{
		Labs:       labService,
		Challenges: challengeService,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to setup grpc server")
	}

	go func() {

		if err = grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("failed to serve")
		}
	}()
	log.Printf("agent gRPC server is running at %s...\n", cfg.GRPC.Endpoint)

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	<-quit

	grpcServer.GracefulStop()
}
