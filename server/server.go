package server

import (
	"context"
	"log"
	"net"

	"github.com/caarlos0/env/v9"
	daprd "github.com/dapr/go-sdk/service/grpc"
	"github.com/shumkovdenis/grpc/graceful"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

type config struct {
	Host string `env:"HOST" envDefault:"127.0.0.1"`
	Port string `env:"PORT" envDefault:"50051"`
}

func (c *config) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
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

	grpcServer := grpc.NewServer()
	pb.RegisterGreeterServer(grpcServer, &server{})

	daprServer := daprd.NewServiceWithGrpcServer(lis, grpcServer)
	graceful.Run(daprServer)
}
