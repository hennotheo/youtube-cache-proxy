package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/hennotheo/youtube-cache-proxy/internal/config"
	"github.com/hennotheo/youtube-cache-proxy/internal/httpapi"
	"github.com/hennotheo/youtube-cache-proxy/internal/server"
	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"github.com/hennotheo/youtube-cache-proxy/internal/worker"
	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	queue, err := store.Open(ctx, settings.SQLitePath)
	if err != nil {
		log.Fatal(err)
	}
	defer queue.Close()
	downloader := youtube.NewDownloader()
	batchWorker, err := worker.New(ctx, queue, downloader, worker.Options{
		AudioDir: settings.AudioDir,
		Limits: youtube.DownloadLimits{
			MaxBytes: int64(settings.MaxAudioBytes), MaxRateBytesPerSecond: int64(settings.MaxRateBytesPerSecond),
			MaxDurationSeconds: int64(settings.MaxVideoDurationSeconds), Timeout: time.Duration(settings.DownloadTimeoutSeconds) * time.Second,
		},
		MaxAttemptsPer24Hours: settings.MaxAttemptsPer24Hours,
		PauseMin:              time.Duration(settings.PauseMinSeconds) * time.Second,
		PauseMax:              time.Duration(settings.PauseMaxSeconds) * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", settings.HTTPAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("HTTP server listening on %s", listener.Addr())
	api := httpapi.NewAPI(queue, batchWorker, downloader, settings.ProducerToken, settings.AdminToken)
	if err := server.Run(ctx, listener, httpapi.NewRouter(api)); err != nil {
		log.Fatal(err)
	}
}
