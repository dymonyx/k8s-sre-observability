package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"chainwise/internal/config"
	"chainwise/internal/frontendui"
	"chainwise/internal/httpx"
	"chainwise/internal/model"
	"chainwise/internal/observability"
)

func main() {
	cfg := config.Load("frontend")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", cfg.ServiceName)
	metrics := observability.New(cfg.ServiceName)
	bikeClient := httpx.NewClient(cfg.BikeAPIURL, cfg.HTTPTimeout, logger)

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/readyz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/check", checkHandler(bikeClient, logger))
	mux.Handle("/", frontendui.Handler())

	handler := httpx.Chain(
		mux,
		httpx.Recover(logger),
		httpx.RequestID(),
		httpx.AccessLog(logger),
		metrics.Middleware,
		httpx.FaultInjection(cfg.DemoLatency, cfg.DemoFailRate, logger),
	)

	server := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := httpx.RunServer(server, logger); err != nil {
		logger.Error("service stopped", "error", err)
		os.Exit(1)
	}
}

func checkHandler(client *httpx.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var response model.BikeCheckResponse
		if err := client.GetJSON(r.Context(), "/bike/check", r.URL.Query(), &response); err != nil {
			logger.Error("bike check failed", "error", err)
			httpx.WriteError(w, http.StatusBadGateway, "bike-api is unavailable")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, response)
	}
}
