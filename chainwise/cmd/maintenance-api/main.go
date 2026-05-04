package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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

	forecasts := []model.ComponentForecast{
		chainLubeForecast(profile, weatherRisk),
		chainReplacementForecast(profile, weatherRisk),
		brakeForecast(profile),
		tireForecast(profile),
		fullServiceForecast(profile, weatherRisk),
	}

	top := highestPriorityForecast(forecasts)
	priority := top.Priority
	action := top.Action
	reason := top.Reason
	if priority == "low" {
		action = "Keep riding and follow the normal maintenance plan"
		reason = fmt.Sprintf("No urgent service is due. Current weather risk is %s and the main wear items are within their planned intervals.", weatherRisk.Risk)
	}

	nextReminder := time.Now().UTC().AddDate(0, 0, 14).Format(time.DateOnly)
	if weatherRisk.Reminder != nil && weatherRisk.Reminder.NextDate != "" {
		nextReminder = weatherRisk.Reminder.NextDate
	}

	return model.MaintenanceRecommendation{
		Bike:              profile.Name,
		Recommendation:    action,
		Priority:          priority,
		Reason:            reason,
		KmSinceService:    kmSinceService,
		KmSinceChainLube:  kmSinceChainLube,
		WeatherRisk:       weatherRisk,
		ComponentForecast: forecasts,
		NextReminder:      nextReminder,
	}
}

func chainLubeForecast(profile model.BikeProfile, weatherRisk model.WeatherRisk) model.ComponentForecast {
	kmSince := max(0, profile.CurrentOdometerKM-profile.LastChainLubeOdometerKM)
	interval := adjustedInterval(baseChainLubeInterval(profile), weatherRisk)
	forecast := intervalForecast("chain_lubrication", "Chain lubrication", kmSince, interval, "Clean and lubricate the chain")

	manualCondition := strings.ToLower(profile.ChainCondition)
	if manualCondition == "dirty" || manualCondition == "dry" || manualCondition == "slightly dry" {
		forecast = forceAtLeast(forecast, "medium", "soon")
		forecast.Reason = fmt.Sprintf("%d km since last lubrication. Chain condition is marked as %s.", kmSince, profile.ChainCondition)
	}
	if manualCondition == "worn" {
		forecast = forceAtLeast(forecast, "high", "due_now")
		forecast.Action = "Clean the drivetrain and measure chain wear"
		forecast.Reason = fmt.Sprintf("Chain is marked as worn and has %d km since last lubrication.", kmSince)
	}
	if weatherRisk.Risk == "medium" && forecast.Priority == "low" {
		forecast = forceAtLeast(forecast, "medium", "soon")
		forecast.Reason = fmt.Sprintf("%d km since last lubrication. Wet or humid weather shortens the lubrication interval.", kmSince)
	}
	if weatherRisk.Risk == "high" {
		forecast = forceAtLeast(forecast, "high", "due_now")
		forecast.Reason = fmt.Sprintf("%d km since last lubrication. Severe weather can wash lubricant away and increase drivetrain wear.", kmSince)
	}
	return forecast
}

func chainReplacementForecast(profile model.BikeProfile, weatherRisk model.WeatherRisk) model.ComponentForecast {
	kmSince := max(0, profile.CurrentOdometerKM-profile.LastChainReplacementOdometerKM)
	interval := adjustedChainReplacementInterval(profile, weatherRisk)
	forecast := intervalForecast("chain_replacement", "Chain replacement", kmSince, interval, "Measure chain wear")
	forecast.Action = "Measure chain wear"
	forecast.Reason = fmt.Sprintf("Estimated %d km since chain replacement. Use a chain checker for an accurate replacement decision.", kmSince)

	switch strings.ToLower(profile.ChainWear) {
	case "0.75% or more", "1.0%", "replace now":
		forecast.Priority = "high"
		forecast.Status = "due_now"
		forecast.RemainingKM = 0
		forecast.OverdueKM = max(0, kmSince-interval)
		forecast.Action = "Replace the chain"
		forecast.Reason = "Measured chain wear is at or above the replacement threshold."
	case "0.5%":
		forecast = forceAtLeast(forecast, "medium", "soon")
		forecast.Action = "Measure again soon"
		forecast.Reason = "Chain wear is measurable. Recheck soon, especially on modern drivetrains."
	case "unknown", "":
		if forecast.Priority == "high" {
			forecast.Status = "measure_needed"
			forecast.Action = "Measure chain wear now"
			forecast.Reason = fmt.Sprintf("Estimated %d km since chain replacement. Exact wear is unknown, so measure before replacing.", kmSince)
		} else if forecast.Priority == "medium" {
			forecast.Status = "measure_needed"
			forecast.Action = "Measure chain wear soon"
		}
	}
	return forecast
}

func brakeForecast(profile model.BikeProfile) model.ComponentForecast {
	kmSince := max(0, profile.CurrentOdometerKM-profile.LastBrakeCheckOdometerKM)
	interval := adjustedByStyle(500, profile)
	forecast := intervalForecast("brakes", "Brake pads", kmSince, interval, "Inspect brake pads")

	symptoms := strings.ToLower(profile.BrakeSymptoms)
	condition := strings.ToLower(profile.BrakeCondition)
	thickness := strings.ToLower(profile.BrakePadThickness)

	if symptoms != "" && symptoms != "none" || condition == "worn" || condition == "weak braking" {
		forecast.Priority = "high"
		forecast.Status = "due_now"
		forecast.Action = "Inspect brakes before riding"
		forecast.Reason = "Brake symptoms or weak braking were reported."
		return forecast
	}
	if thickness == "less than 1 mm" || thickness == "<1 mm" {
		forecast.Priority = "high"
		forecast.Status = "due_now"
		forecast.Action = "Replace brake pads"
		forecast.Reason = "Brake pad material is reported below 1 mm."
		return forecast
	}
	if thickness == "1-2 mm" || thickness == "1–2 mm" || condition == "check soon" {
		forecast = forceAtLeast(forecast, "medium", "soon")
		forecast.Action = "Inspect brake pads soon"
		forecast.Reason = "Brake pads are not critical yet, but they should be inspected soon."
	}
	if forecast.Priority != "low" && thickness == "unknown" && symptoms == "none" {
		forecast.Reason = fmt.Sprintf("%d km since brake check. Pad thickness is unknown, so inspect instead of replacing blindly.", kmSince)
	}
	return forecast
}

func tireForecast(profile model.BikeProfile) model.ComponentForecast {
	kmSince := max(0, profile.CurrentOdometerKM-profile.LastTireCheckOdometerKM)
	interval := adjustedByStyle(250, profile)
	forecast := intervalForecast("tires", "Tires", kmSince, interval, "Check tire pressure and tread")

	condition := strings.ToLower(profile.TireCondition)
	if condition == "cracked" || condition == "cuts" || condition == "worn tread" || condition == "worn" {
		forecast.Priority = "high"
		forecast.Status = "due_now"
		forecast.Action = "Inspect tires before riding"
		forecast.Reason = "Tire damage or worn tread was reported."
		return forecast
	}
	if condition == "low pressure" || pressureLooksLow(profile) {
		forecast = forceAtLeast(forecast, "medium", "due_now")
		forecast.Action = "Inflate tires before riding"
		forecast.Reason = "Tire pressure is low or below the expected range for this bike type."
	}
	if profile.RecentPunctures >= 2 {
		forecast = forceAtLeast(forecast, "medium", "soon")
		forecast.Action = "Inspect tire casing and rim tape"
		forecast.Reason = "Two or more recent punctures can indicate tire damage or rim tape problems."
	}
	if forecast.Priority != "low" && profile.FrontTirePressureBar == 0 && profile.RearTirePressureBar == 0 && condition == "unknown" {
		forecast.Reason = fmt.Sprintf("%d km since tire check. Tire pressure and condition were not provided, so a quick pre-ride check is recommended.", kmSince)
	}
	return forecast
}

func fullServiceForecast(profile model.BikeProfile, weatherRisk model.WeatherRisk) model.ComponentForecast {
	kmSince := max(0, profile.CurrentOdometerKM-profile.LastServiceOdometerKM)
	interval := adjustedByStyle(1000, profile)
	if weatherRisk.Risk == "high" {
		interval = int(float64(interval) * 0.85)
	}
	forecast := intervalForecast("full_service", "Full service", kmSince, interval, "Schedule a full bike inspection")
	if forecast.Priority != "low" {
		forecast.Reason = fmt.Sprintf("%d km since full service. A broader inspection is recommended for drivetrain, brakes, tires and bolts.", kmSince)
	}
	return forecast
}

func intervalForecast(component, label string, kmSince int, interval int, action string) model.ComponentForecast {
	remaining := interval - kmSince
	forecast := model.ComponentForecast{
		Component:   component,
		Label:       label,
		Status:      "ok",
		Priority:    "low",
		KmSince:     kmSince,
		IntervalKM:  interval,
		RemainingKM: max(0, remaining),
		OverdueKM:   max(0, -remaining),
		Action:      action,
		Reason:      fmt.Sprintf("%d km since last check. Planned interval is %d km.", kmSince, interval),
	}

	soonThreshold := max(40, interval/5)
	if remaining <= -soonThreshold {
		forecast.Status = "overdue"
		forecast.Priority = "high"
		forecast.Reason = fmt.Sprintf("%d km since last check, about %d km overdue.", kmSince, forecast.OverdueKM)
	} else if remaining <= 0 {
		forecast.Status = "due_now"
		forecast.Priority = "high"
		forecast.Reason = fmt.Sprintf("%d km since last check. Planned interval has been reached.", kmSince)
	} else if remaining <= soonThreshold {
		forecast.Status = "soon"
		forecast.Priority = "medium"
		forecast.Reason = fmt.Sprintf("%d km since last check. About %d km remaining.", kmSince, remaining)
	}

	return forecast
}

func baseChainLubeInterval(profile model.BikeProfile) int {
	switch strings.ToLower(profile.Type) {
	case "mountain bike", "mtb":
		return adjustedByStyle(120, profile)
	case "gravel bike":
		return adjustedByStyle(140, profile)
	case "road bike":
		return adjustedByStyle(180, profile)
	case "city bike", "hybrid bike":
		return adjustedByStyle(170, profile)
	default:
		return adjustedByStyle(160, profile)
	}
}

func adjustedChainReplacementInterval(profile model.BikeProfile, weatherRisk model.WeatherRisk) int {
	base := 2400
	switch strings.ToLower(profile.Type) {
	case "mountain bike", "mtb":
		base = 1600
	case "gravel bike":
		base = 2000
	case "road bike":
		base = 2600
	case "city bike", "hybrid bike":
		base = 2400
	}
	base = adjustedByStyle(base, profile)
	if weatherRisk.Risk == "medium" {
		base = int(float64(base) * 0.9)
	}
	if weatherRisk.Risk == "high" {
		base = int(float64(base) * 0.8)
	}
	return max(800, base)
}

func adjustedInterval(base int, weatherRisk model.WeatherRisk) int {
	if weatherRisk.Risk == "high" {
		return max(40, int(float64(base)*0.45))
	}
	if weatherRisk.Risk == "medium" {
		return max(60, int(float64(base)*0.65))
	}
	return base
}

func adjustedByStyle(base int, profile model.BikeProfile) int {
	switch strings.ToLower(profile.RidingStyle) {
	case "sport training":
		return int(float64(base) * 0.9)
	case "gravel or muddy rides":
		return int(float64(base) * 0.8)
	case "daily commuting":
		return int(float64(base) * 0.9)
	default:
		return base
	}
}

func pressureLooksLow(profile model.BikeProfile) bool {
	if profile.FrontTirePressureBar == 0 && profile.RearTirePressureBar == 0 {
		return false
	}

	minimum := 3.0
	switch strings.ToLower(profile.Type) {
	case "road bike":
		minimum = 5.0
	case "gravel bike":
		minimum = 2.0
	case "mountain bike", "mtb":
		minimum = 1.4
	case "city bike", "hybrid bike":
		minimum = 3.0
	}

	return profile.FrontTirePressureBar > 0 && profile.FrontTirePressureBar < minimum || profile.RearTirePressureBar > 0 && profile.RearTirePressureBar < minimum
}

func forceAtLeast(forecast model.ComponentForecast, priority string, status string) model.ComponentForecast {
	if priorityRank(priority) > priorityRank(forecast.Priority) {
		forecast.Priority = priority
		forecast.Status = status
		if status == "due_now" {
			forecast.RemainingKM = 0
		}
	}
	return forecast
}

func highestPriorityForecast(forecasts []model.ComponentForecast) model.ComponentForecast {
	if len(forecasts) == 0 {
		return model.ComponentForecast{Priority: "low", Action: "Keep riding", Reason: "No forecast available."}
	}
	best := forecasts[0]
	for _, forecast := range forecasts[1:] {
		if priorityRank(forecast.Priority) > priorityRank(best.Priority) {
			best = forecast
			continue
		}
		if priorityRank(forecast.Priority) == priorityRank(best.Priority) && statusRank(forecast.Status) > statusRank(best.Status) {
			best = forecast
		}
	}
	return best
}

func priorityRank(priority string) int {
	switch priority {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func statusRank(status string) int {
	switch status {
	case "overdue":
		return 5
	case "due_now":
		return 4
	case "measure_needed":
		return 3
	case "soon":
		return 2
	default:
		return 1
	}
}

func bikeProfileFromQuery(query url.Values) model.BikeProfile {
	currentOdometer := parseNonNegativeInt(query.Get("currentOdometerKm"), 1240)
	lastServiceOdometer := parseNonNegativeInt(query.Get("lastServiceOdometerKm"), max(0, currentOdometer-260))
	lastChainLubeOdometer := parseNonNegativeInt(query.Get("lastChainLubeOdometerKm"), max(0, currentOdometer-80))
	lastChainReplacementOdometer := parseNonNegativeInt(query.Get("lastChainReplacementOdometerKm"), max(0, currentOdometer-900))
	lastBrakeCheckOdometer := parseNonNegativeInt(query.Get("lastBrakeCheckOdometerKm"), max(0, currentOdometer-300))
	lastTireCheckOdometer := parseNonNegativeInt(query.Get("lastTireCheckOdometerKm"), max(0, currentOdometer-120))

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
