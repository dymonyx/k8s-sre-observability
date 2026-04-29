package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func RunServer(server *http.Server, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("service started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logger.Info("service shutting down")
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func HealthHandler(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"service": service, "status": "ok"})
	}
}
