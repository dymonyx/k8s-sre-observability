package main

import (
	"fmt"
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
	cfg := config.Load("maintenance-api")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", cfg.ServiceName)
	metrics := observability.New(cfg.ServiceName)
	weatherClient := httpx.NewClient(cfg.WeatherAPIURL, cfg.HTTPTimeout, logger)

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/readyz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/maintenance/recommendation", recommendationHandler(weatherClient, logger))

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

func recommendationHandler(client *httpx.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := bikeProfileFromQuery(r.URL.Query())

		weatherQuery := url.Values{}
		weatherQuery.Set("maintenanceType", "chain_lubrication")

		var weatherRisk model.WeatherRisk
		if err := client.GetJSON(r.Context(), "/weather/risk", weatherQuery, &weatherRisk); err != nil {
			logger.Error("weather risk failed", "error", err)
			httpx.WriteError(w, http.StatusBadGateway, "weather-api is unavailable")
			return
		}

		recommendation := calculateRecommendation(profile, weatherRisk)
		httpx.WriteJSON(w, http.StatusOK, recommendation)
	}
}

func calculateRecommendation(profile model.BikeProfile, weatherRisk model.WeatherRisk) model.MaintenanceRecommendation {
	kmSinceService := max(0, profile.CurrentOdometerKM-profile.LastServiceOdometerKM)
	kmSinceChainLube := max(0, profile.CurrentOdometerKM-profile.LastChainLubeOdometerKM)

	priority := "low"
	action := "Inspect the bike and keep the normal maintenance schedule"
	reason := fmt.Sprintf("%s has %d km since the last chain lubrication and %d km since the last service.", profile.Name, kmSinceChainLube, kmSinceService)

	chainNeedsCare := profile.ChainCondition == "slightly dry" || profile.ChainCondition == "dirty" || profile.ChainCondition == "worn"
	brakesNeedCare := profile.BrakeCondition == "check soon" || profile.BrakeCondition == "worn"
	tiresNeedCare := profile.TireCondition == "low pressure" || profile.TireCondition == "worn"

	if kmSinceChainLube >= 100 || weatherRisk.Risk == "medium" || chainNeedsCare {
		priority = "medium"
		action = "Clean and lubricate the chain"
		reason = fmt.Sprintf("You rode %d km since the last chain lubrication. Current weather risk is %s (%s), and chain condition is %s.", kmSinceChainLube, weatherRisk.Risk, weatherRisk.Condition, profile.ChainCondition)
	}

	if kmSinceService >= 500 && priority == "low" {
		priority = "medium"
		action = "Schedule a general bike inspection"
		reason = fmt.Sprintf("You rode %d km since the last service. A general inspection is recommended soon.", kmSinceService)
	}

	if weatherRisk.Risk == "high" || kmSinceChainLube >= 180 || profile.ChainCondition == "worn" || profile.BrakeCondition == "worn" || profile.TireCondition == "worn" {
		priority = "high"
		action = "Clean the drivetrain and inspect safety-critical components"
		reason = fmt.Sprintf("High maintenance risk: %d km since chain lubrication, %d km since service, weather risk %s, brakes %s, tires %s.", kmSinceChainLube, kmSinceService, weatherRisk.Risk, profile.BrakeCondition, profile.TireCondition)
	}

	if brakesNeedCare && priority != "high" {
		priority = "medium"
		action = "Inspect brake pads and clean the drivetrain"
		reason = fmt.Sprintf("Brake condition is %s. You also rode %d km since the last service.", profile.BrakeCondition, kmSinceService)
	}

	if tiresNeedCare && priority != "high" {
		priority = "medium"
		action = "Check tire pressure and inspect tire wear"
		reason = fmt.Sprintf("Tire condition is %s. Check tire pressure before the next ride.", profile.TireCondition)
	}

	nextReminder := time.Now().UTC().AddDate(0, 0, 14).Format(time.DateOnly)
	if weatherRisk.Reminder != nil && weatherRisk.Reminder.NextDate != "" {
		nextReminder = weatherRisk.Reminder.NextDate
	}

	return model.MaintenanceRecommendation{
		Bike:             profile.Name,
		Recommendation:   action,
		Priority:         priority,
		Reason:           reason,
		KmSinceService:   kmSinceService,
		KmSinceChainLube: kmSinceChainLube,
		WeatherRisk:      weatherRisk,
		NextReminder:     nextReminder,
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

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
