package graceful

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

type Service interface {
	Start() error
	GracefulStop() error
}

func Run(service Service) {
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
