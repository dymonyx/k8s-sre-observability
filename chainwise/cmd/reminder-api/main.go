package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	cfg := config.Load("reminder-api")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", cfg.ServiceName)
	metrics := observability.New(cfg.ServiceName)
	userClient := httpx.NewClient(cfg.UserAPIURL, cfg.HTTPTimeout, logger)

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/readyz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/reminders/next", nextReminderHandler(userClient, logger))
	mux.HandleFunc("/reminders", createReminderHandler)

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

func nextReminderHandler(client *httpx.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maintenanceType := firstNonEmpty(r.URL.Query().Get("type"), "chain_lubrication")
		risk := firstNonEmpty(r.URL.Query().Get("risk"), "medium")

		var preferences model.UserPreferences
		if err := client.GetJSON(r.Context(), "/users/preferences", nil, &preferences); err != nil {
			logger.Error("user preferences failed", "error", err)
			httpx.WriteError(w, http.StatusBadGateway, "user-api is unavailable")
			return
		}

		reminder := calculateReminder(maintenanceType, risk, preferences)
		httpx.WriteJSON(w, http.StatusOK, reminder)
	}
}

func createReminderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		httpx.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "could not read request body")
		return
	}

	var reminder model.Reminder
	if err := json.Unmarshal(body, &reminder); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid reminder payload")
		return
	}
	if reminder.NextDate == "" {
		reminder.NextDate = time.Now().UTC().AddDate(0, 0, 7).Format(time.DateOnly)
	}
	if reminder.Message == "" {
		reminder.Message = "Maintenance reminder created"
	}

	httpx.WriteJSON(w, http.StatusCreated, reminder)
}

func calculateReminder(maintenanceType string, risk string, preferences model.UserPreferences) model.Reminder {
	priority := priorityFromRisk(risk)
	days := daysUntilNextReminder(priority, preferences)
	nextDate := time.Now().UTC().AddDate(0, 0, days).Format(time.DateOnly)

	return model.Reminder{
		Type:     maintenanceType,
		Priority: priority,
		NextDate: nextDate,
		Channel:  preferences.NotificationChannel,
		Message:  fmt.Sprintf("%s reminder scheduled based on %s weather risk", maintenanceType, risk),
	}
}

func priorityFromRisk(risk string) string {
	switch risk {
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func daysUntilNextReminder(priority string, preferences model.UserPreferences) int {
	switch priority {
	case "high":
		if preferences.MaintenanceExperience == "beginner" {
			return 2
		}
		return 3
	case "medium":
		return 5
	default:
		return 14
	}
}

func firstNonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
