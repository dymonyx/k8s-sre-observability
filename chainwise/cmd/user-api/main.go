package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"chainwise/internal/config"
	"chainwise/internal/httpx"
	"chainwise/internal/model"
	"chainwise/internal/observability"
)

func main() {
	cfg := config.Load("user-api")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", cfg.ServiceName)
	metrics := observability.New(cfg.ServiceName)

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/readyz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/users/demo", demoUserHandler)
	mux.HandleFunc("/users/preferences", preferencesHandler)

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

func demoUserHandler(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, model.User{
		ID:    "user-demo-001",
		Name:  "Demo Cyclist",
		Level: "beginner",
	})
}

func preferencesHandler(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, model.UserPreferences{
		UserID:                  "user-demo-001",
		PreferredReminderEvery:  "weekly",
		NotificationChannel:     "email",
		AverageWeeklyDistanceKM: 80,
		MaintenanceExperience:   "beginner",
		BikeUsageType:           "commute and weekend rides",
	})
}
