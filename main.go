package main

import (
	"flag"

	"github.com/shumkovdenis/grpc/client"
	"github.com/shumkovdenis/grpc/server"
)

func main() {
	httpFlag := flag.Bool("client", false, "Start client")
	flag.Parse()

	if *httpFlag {
		client.Start()
	} else {
		server.Start()
	}
}
