package server

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v9"
	daprd "github.com/dapr/go-sdk/service/grpc"
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

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	daprServer := daprd.NewServiceWithGrpcServer(lis, s)

	run(daprServer)
}

type Service interface {
	Start() error
	GracefulStop() error
}

func run(service Service) {
	errChan := make(chan error)
	stopChan := make(chan os.Signal, 1)

	// bind OS events to the signal channel
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)

	// run blocking call in a separate goroutine, report errors via channel
	go func() {
		log.Println("starting server")

		if err := service.Start(); err != nil {
			errChan <- err
		}
	}()

	// terminate your environment gracefully before leaving main function
	defer func() {
		log.Println("stopping server")

		if err := service.GracefulStop(); err != nil {
			log.Fatalf("graceful stop failed: %v", err)
		}
	}()

	// block until either OS signal, or server fatal error
	select {
	case err := <-errChan:
		log.Fatalf("server error: %v", err)
	case <-stopChan:
	}
}
