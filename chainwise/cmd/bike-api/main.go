package main

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"chainwise/internal/config"
	"chainwise/internal/httpx"
	"chainwise/internal/model"
	"chainwise/internal/observability"
)

func main() {
	cfg := config.Load("bike-api")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", cfg.ServiceName)
	metrics := observability.New(cfg.ServiceName)
	maintenanceClient := httpx.NewClient(cfg.MaintenanceAPIURL, cfg.HTTPTimeout, logger)

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/readyz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/bike/profile", profileHandler)
	mux.HandleFunc("/bike/check", checkHandler(maintenanceClient, logger))

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

func profileHandler(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, bikeProfileFromQuery(r.URL.Query()))
}

func checkHandler(client *httpx.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := bikeProfileFromQuery(r.URL.Query())

		query := url.Values{}
		query.Set("bikeName", profile.Name)
		query.Set("bikeType", profile.Type)
		query.Set("currentOdometerKm", strconv.Itoa(profile.CurrentOdometerKM))
		query.Set("lastRideDistanceKm", strconv.Itoa(profile.LastRideDistanceKM))
		query.Set("lastRideDate", profile.LastRideDate)
		query.Set("lastServiceDate", profile.LastServiceDate)
		query.Set("lastServiceOdometerKm", strconv.Itoa(profile.LastServiceOdometerKM))
		query.Set("lastChainLubeOdometerKm", strconv.Itoa(profile.LastChainLubeOdometerKM))
		query.Set("ridingStyle", profile.RidingStyle)
		query.Set("chainCondition", profile.ChainCondition)
		query.Set("brakeCondition", profile.BrakeCondition)
		query.Set("tireCondition", profile.TireCondition)

		var recommendation model.MaintenanceRecommendation
		if err := client.GetJSON(r.Context(), "/maintenance/recommendation", query, &recommendation); err != nil {
			logger.Error("maintenance recommendation failed", "error", err)
			httpx.WriteError(w, http.StatusBadGateway, "maintenance-api is unavailable")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, model.BikeCheckResponse{
			BikeProfile:    profile,
			Recommendation: recommendation,
		})
	}
}

func bikeProfileFromQuery(query url.Values) model.BikeProfile {
	currentOdometer := parseNonNegativeInt(query.Get("currentOdometerKm"), 1240)
	lastServiceOdometer := parseNonNegativeInt(query.Get("lastServiceOdometerKm"), 980)
	lastChainLubeOdometer := parseNonNegativeInt(query.Get("lastChainLubeOdometerKm"), 1160)

	if lastServiceOdometer > currentOdometer {
		lastServiceOdometer = currentOdometer
	}
	if lastChainLubeOdometer > currentOdometer {
		lastChainLubeOdometer = currentOdometer
	}

	return model.BikeProfile{
		ID:                      "bike-demo-001",
		Name:                    firstNonEmpty(query.Get("bikeName"), "My Gravel Bike"),
		Type:                    firstNonEmpty(query.Get("bikeType"), "gravel bike"),
		CurrentOdometerKM:       currentOdometer,
		LastRideDistanceKM:      parseNonNegativeInt(query.Get("lastRideDistanceKm"), 0),
		LastRideDate:            firstNonEmpty(query.Get("lastRideDate"), time.Now().UTC().Format(time.DateOnly)),
		LastServiceDate:         firstNonEmpty(query.Get("lastServiceDate"), time.Now().UTC().AddDate(0, 0, -18).Format(time.DateOnly)),
		LastServiceOdometerKM:   lastServiceOdometer,
		LastChainLubeOdometerKM: lastChainLubeOdometer,
		RidingStyle:             firstNonEmpty(query.Get("ridingStyle"), "commute and weekend rides"),
		ChainCondition:          firstNonEmpty(query.Get("chainCondition"), "slightly dry"),
		BrakeCondition:          firstNonEmpty(query.Get("brakeCondition"), "good"),
		TireCondition:           firstNonEmpty(query.Get("tireCondition"), "good"),
	}
}

func parseNonNegativeInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func firstNonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
