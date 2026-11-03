package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"

	"github.com/hennotheo/youtube-cache-proxy/internal/config"
	"github.com/hennotheo/youtube-cache-proxy/internal/httpapi"
	"github.com/hennotheo/youtube-cache-proxy/internal/server"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	listener, err := net.Listen("tcp", settings.HTTPAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("HTTP server listening on %s", listener.Addr())
	if err := server.Run(ctx, listener, httpapi.NewRouter()); err != nil {
		log.Fatal(err)
	}
}
