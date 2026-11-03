package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"

	"github.com/hennotheo/youtube-cache-proxy/internal/httpapi"
	"github.com/hennotheo/youtube-cache-proxy/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("HTTP server listening on %s", listener.Addr())
	if err := server.Run(ctx, listener, httpapi.NewRouter()); err != nil {
		log.Fatal(err)
	}
}
