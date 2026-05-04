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
		query.Set("lastChainReplacementOdometerKm", strconv.Itoa(profile.LastChainReplacementOdometerKM))
		query.Set("lastBrakeCheckOdometerKm", strconv.Itoa(profile.LastBrakeCheckOdometerKM))
		query.Set("lastTireCheckOdometerKm", strconv.Itoa(profile.LastTireCheckOdometerKM))
		query.Set("ridingStyle", profile.RidingStyle)
		query.Set("chainCondition", profile.ChainCondition)
		query.Set("chainWear", profile.ChainWear)
		query.Set("brakeCondition", profile.BrakeCondition)
		query.Set("brakePadThickness", profile.BrakePadThickness)
		query.Set("brakeSymptoms", profile.BrakeSymptoms)
		query.Set("tireCondition", profile.TireCondition)
		query.Set("recentPunctures", strconv.Itoa(profile.RecentPunctures))
		query.Set("frontTirePressureBar", formatFloat(profile.FrontTirePressureBar))
		query.Set("rearTirePressureBar", formatFloat(profile.RearTirePressureBar))

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
	currentOdometer := parseNonNegativeInt(query.Get("currentOdometerKm"), 0)
	lastServiceOdometer := parseNonNegativeInt(query.Get("lastServiceOdometerKm"), 0)
	lastChainLubeOdometer := parseNonNegativeInt(query.Get("lastChainLubeOdometerKm"), 0)
	lastChainReplacementOdometer := parseNonNegativeInt(query.Get("lastChainReplacementOdometerKm"), 0)
	lastBrakeCheckOdometer := parseNonNegativeInt(query.Get("lastBrakeCheckOdometerKm"), 0)
	lastTireCheckOdometer := parseNonNegativeInt(query.Get("lastTireCheckOdometerKm"), 0)

	lastServiceOdometer = clampMax(lastServiceOdometer, currentOdometer)
	lastChainLubeOdometer = clampMax(lastChainLubeOdometer, currentOdometer)
	lastChainReplacementOdometer = clampMax(lastChainReplacementOdometer, currentOdometer)
	lastBrakeCheckOdometer = clampMax(lastBrakeCheckOdometer, currentOdometer)
	lastTireCheckOdometer = clampMax(lastTireCheckOdometer, currentOdometer)

	return model.BikeProfile{
		ID:                             "bike-demo-001",
		Name:                           firstNonEmpty(query.Get("bikeName"), "My Gravel Bike"),
		Type:                           firstNonEmpty(query.Get("bikeType"), "gravel bike"),
		CurrentOdometerKM:              currentOdometer,
		LastRideDistanceKM:             parseNonNegativeInt(query.Get("lastRideDistanceKm"), 0),
		LastRideDate:                   firstNonEmpty(query.Get("lastRideDate"), time.Now().UTC().Format(time.DateOnly)),
		LastServiceDate:                firstNonEmpty(query.Get("lastServiceDate"), time.Now().UTC().AddDate(0, 0, -18).Format(time.DateOnly)),
		LastServiceOdometerKM:          lastServiceOdometer,
		LastChainLubeOdometerKM:        lastChainLubeOdometer,
		LastChainReplacementOdometerKM: lastChainReplacementOdometer,
		LastBrakeCheckOdometerKM:       lastBrakeCheckOdometer,
		LastTireCheckOdometerKM:        lastTireCheckOdometer,
		RidingStyle:                    firstNonEmpty(query.Get("ridingStyle"), "daily commuting"),
		ChainCondition:                 firstNonEmpty(query.Get("chainCondition"), "unknown"),
		ChainWear:                      firstNonEmpty(query.Get("chainWear"), "unknown"),
		BrakeCondition:                 firstNonEmpty(query.Get("brakeCondition"), "unknown"),
		BrakePadThickness:              firstNonEmpty(query.Get("brakePadThickness"), "unknown"),
		BrakeSymptoms:                  firstNonEmpty(query.Get("brakeSymptoms"), "none"),
		TireCondition:                  firstNonEmpty(query.Get("tireCondition"), "unknown"),
		RecentPunctures:                parseNonNegativeInt(query.Get("recentPunctures"), 0),
		FrontTirePressureBar:           parseNonNegativeFloat(query.Get("frontTirePressureBar"), 0),
		RearTirePressureBar:            parseNonNegativeFloat(query.Get("rearTirePressureBar"), 0),
	}
}

func parseNonNegativeInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func parseNonNegativeFloat(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func firstNonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func clampMax(value int, maximum int) int {
	if value > maximum {
		return maximum
	}
	return value
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
