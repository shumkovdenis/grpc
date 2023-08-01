package client

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/caarlos0/env/v9"
	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/shumkovdenis/grpc/graceful"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/metadata"
)

const retryPolicy = `{
	"methodConfig": [{
	  "name": [{}],
	  "waitForReady": true,
	  "retryPolicy": {
		  "MaxAttempts": 4,
		  "InitialBackoff": ".1s",
		  "MaxBackoff": "1s",
		  "BackoffMultiplier": 2.0,
		  "RetryableStatusCodes": [ "UNAVAILABLE" ]
	  }
	}]}`

type serviceConfig struct {
	Host    string        `env:"HOST" envDefault:"127.0.0.1"`
	Port    string        `env:"PORT" envDefault:"${DAPR_GRPC_PORT}" envExpand:"true"`
	Name    string        `env:"NAME" envDefault:"grpc-server"`
	Timeout time.Duration `env:"TIMEOUT" envDefault:"5s"`
}

func (c *serviceConfig) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

type config struct {
	Host         string        `env:"HOST" envDefault:""`
	Port         string        `env:"PORT" envDefault:"3000"`
	Service      serviceConfig `envPrefix:"SERVICE_"`
	Sleep        time.Duration `env:"SLEEP" envDefault:"3s"`
	Autostart    bool          `env:"AUTOSTART" envDefault:"true"`
	SkipResponse bool          `env:"SKIP_RESPONSE" envDefault:"true"`
	WithBlock    bool          `env:"WITH_BLOCK" envDefault:"false"`
	WithBalancer bool          `env:"WITH_BALANCER" envDefault:"false"`
	WaitForReady bool          `env:"WAIT_FOR_READY" envDefault:"false"`
	WithRetry    bool          `env:"WITH_RETRY" envDefault:"false"`
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

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if cfg.WithBlock {
		opts = append(opts, grpc.WithBlock())
	}

	if cfg.WithBalancer {
		opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`))
	}

	if cfg.WithRetry {
		// opts = append(opts, grpc.WithDefaultServiceConfig(retryPolicy))
		opts = append(
			opts,
			grpc.WithUnaryInterceptor(
				retry.UnaryClientInterceptor(
					retry.WithMax(5),
					retry.WithOnRetryCallback(func(ctx context.Context, attempt uint, err error) {
						log.Printf("grpc_retry attempt: %d, backoff for %v", attempt, err)
					}),
				),
			),
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Service.Timeout)

	conn, err := grpc.DialContext(ctx, cfg.Service.Address(), opts...)
	cancel()
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	srv := &server{
		cfg:  &cfg,
		conn: conn,
	}

	if cfg.Autostart {
		srv.Toggle()
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/toggle", func(w http.ResponseWriter, r *http.Request) {
		val := srv.Toggle()
		w.Write([]byte("toggled: " + strconv.FormatBool(val)))
	})

	daprServer := daprd.NewServiceWithMux(cfg.Address(), r)
	graceful.Run(daprServer)
}

type server struct {
	cfg    *config
	conn   *grpc.ClientConn
	active bool
	pause  bool
}

func (s *server) Toggle() bool {
	if s.active {
		s.pause = !s.pause
		return s.pause
	}

	s.active = true
	s.pause = false

	client := pb.NewGreeterClient(s.conn)

	go func() {
		for !s.pause {
			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Service.Timeout)

			ctx = metadata.AppendToOutgoingContext(ctx, "dapr-app-id", s.cfg.Service.Name)

			r, err := client.SayHello(ctx, &pb.HelloRequest{Name: "Dapr"}, grpc.WaitForReady(s.cfg.WaitForReady))
			cancel()
			if err != nil {
				log.Printf("could not greet: %v", err)
			} else if !s.cfg.SkipResponse {
				log.Printf("Greeting: %s", r.GetMessage())
			}

			time.Sleep(s.cfg.Sleep)
		}
	}()

	return s.pause
}
