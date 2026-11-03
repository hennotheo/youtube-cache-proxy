package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// Run serves HTTP until the context is cancelled or the listener fails.
func Run(ctx context.Context, listener net.Listener, handler http.Handler) error {
	srv := &http.Server{Handler: handler}
	serveResult := make(chan error, 1)
	go func() { serveResult <- srv.Serve(listener) }()

	select {
	case err := <-serveResult:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			_ = srv.Close()
			return err
		}
		err := <-serveResult
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
