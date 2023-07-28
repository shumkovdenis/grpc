package client

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/metadata"
)

type config struct {
	Host    string        `env:"SERVICE_HOST" envDefault:"127.0.0.1"`
	Port    string        `env:"SERVICE_PORT" envDefault:"${DAPR_GRPC_PORT}" envExpand:"true"`
	Name    string        `env:"SERVICE_NAME" envDefault:"grpc-server"`
	Timeout time.Duration `env:"TIMEOUT" envDefault:"5s"`
	Sleep   time.Duration `env:"SLEEP" envDefault:"3s"`
}

func (c *config) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

func Start() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("%+v", err)
	}

	log.Printf("config %+v", cfg)

	conn, err := grpc.Dial(cfg.Address(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	run(&cfg, conn)
}

func run(cfg *config, conn *grpc.ClientConn) {
	errChan := make(chan error)
	stopChan := make(chan os.Signal, 1)

	// bind OS events to the signal channel
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)

	// run blocking call in a separate goroutine, report errors via channel
	go func() {
		log.Println("starting server")

		calls(cfg, pb.NewGreeterClient(conn))
	}()

	// terminate your environment gracefully before leaving main function
	defer func() {
		log.Println("stopping server")

		conn.Close()
	}()

	// block until either OS signal, or server fatal error
	select {
	case err := <-errChan:
		log.Fatalf("server error: %v", err)
	case <-stopChan:
	}
}

func calls(cfg *config, client pb.GreeterClient) {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)

		ctx = metadata.AppendToOutgoingContext(ctx, "dapr-app-id", cfg.Name)

		r, err := client.SayHello(ctx, &pb.HelloRequest{Name: "Dapr"})
		cancel()
		if err != nil {
			log.Printf("could not greet: %v", err)
		}

		log.Printf("Greeting: %s", r.GetMessage())

		time.Sleep(cfg.Sleep)
	}
}
