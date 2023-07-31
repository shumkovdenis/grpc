package server

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/caarlos0/env/v9"
	daprd "github.com/dapr/go-sdk/service/grpc"
	"github.com/shumkovdenis/grpc/graceful"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

type config struct {
	Host        string        `env:"HOST" envDefault:""`
	Port        string        `env:"PORT" envDefault:"50051"`
	Sleep       time.Duration `env:"SLEEP" envDefault:"1s"`
	SkipRequest bool          `env:"SKIP_REQUEST" envDefault:"true"`
	SkipHealth  bool          `env:"SKIP_HEALTH" envDefault:"true"`
	WithTimeout bool          `env:"WITH_TIMEOUT" envDefault:"false"`
}

func (c *config) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

type server struct {
	pb.UnimplementedGreeterServer
	cfg *config
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	if !s.cfg.SkipRequest {
		log.Printf("Received: %v", in.GetName())
	}

	time.Sleep(s.cfg.Sleep)

	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func Start() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("%+v", err)
	}

	log.Printf("config %+v", cfg)

	lis, err := net.Listen("tcp", cfg.Address())
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}

	if cfg.WithTimeout {
		opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     time.Second,
			MaxConnectionAge:      time.Second,
			MaxConnectionAgeGrace: time.Second,
			Time:                  time.Second * 10,
			Timeout:               time.Second,
		}))
	}

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterGreeterServer(grpcServer, &server{cfg: &cfg})
	grpc_health_v1.RegisterHealthServer(grpcServer, &health{cfg: &cfg})

	daprServer := daprd.NewServiceWithGrpcServer(lis, grpcServer)
	graceful.Run(daprServer)
}

type health struct {
	grpc_health_v1.UnimplementedHealthServer
	cfg *config
}

func (h health) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	if !h.cfg.SkipHealth {
		log.Println("serving health")
	}
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}
