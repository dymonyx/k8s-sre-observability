package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"chainwise/internal/config"
	"chainwise/internal/httpx"
	"chainwise/internal/model"
	"chainwise/internal/observability"
)

const openMeteoForecastURL = "https://api.open-meteo.com/v1/forecast"

type weatherService struct {
	client    *http.Client
	city      string
	latitude  float64
	longitude float64
}

type openMeteoResponse struct {
	Current struct {
		TemperatureC         float64 `json:"temperature_2m"`
		ApparentTemperatureC float64 `json:"apparent_temperature"`
		Humidity             int     `json:"relative_humidity_2m"`
		PrecipitationMM      float64 `json:"precipitation"`
		RainMM               float64 `json:"rain"`
		SnowfallCM           float64 `json:"snowfall"`
		WeatherCode          int     `json:"weather_code"`
		WindSpeedMS          float64 `json:"wind_speed_10m"`
		WindGustsMS          float64 `json:"wind_gusts_10m"`
	} `json:"current"`
}

func main() {
	cfg := config.Load("weather-api")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", cfg.ServiceName)
	metrics := observability.New(cfg.ServiceName)

	reminderClient := httpx.NewClient(cfg.ReminderAPIURL, cfg.HTTPTimeout, logger)
	weatherClient := &weatherService{
		client: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		city:      cfg.WeatherCity,
		latitude:  cfg.WeatherLatitude,
		longitude: cfg.WeatherLongitude,
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/readyz", httpx.HealthHandler(cfg.ServiceName))
	mux.HandleFunc("/weather/current", currentHandler(weatherClient, logger))
	mux.HandleFunc("/weather/risk", riskHandler(weatherClient, reminderClient, logger))

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

func currentHandler(service *weatherService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		weather, err := service.currentWeather(r.Context())
		if err != nil {
			logger.Warn("open-meteo unavailable, using fallback weather", "error", err)
			weather = service.fallbackWeather()
		}

		httpx.WriteJSON(w, http.StatusOK, weather)
	}
}

func riskHandler(service *weatherService, client *httpx.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maintenanceType := firstNonEmpty(r.URL.Query().Get("maintenanceType"), "chain_lubrication")

		weather, err := service.currentWeather(r.Context())
		if err != nil {
			logger.Warn("open-meteo unavailable, using fallback weather", "error", err)
			weather = service.fallbackWeather()
		}

		risk := riskFromWeather(weather)

		query := url.Values{}
		query.Set("type", maintenanceType)
		query.Set("risk", risk.Risk)

		var reminder model.Reminder
		if err := client.GetJSON(r.Context(), "/reminders/next", query, &reminder); err != nil {
			logger.Error("reminder calculation failed", "error", err)
			httpx.WriteError(w, http.StatusBadGateway, "reminder-api is unavailable")
			return
		}

		risk.Reminder = &reminder
		httpx.WriteJSON(w, http.StatusOK, risk)
	}
}

func (s *weatherService) currentWeather(ctx context.Context) (model.WeatherCurrent, error) {
	endpoint, err := url.Parse(openMeteoForecastURL)
	if err != nil {
		return model.WeatherCurrent{}, fmt.Errorf("parse open-meteo url: %w", err)
	}

	query := endpoint.Query()
	query.Set("latitude", fmt.Sprintf("%.4f", s.latitude))
	query.Set("longitude", fmt.Sprintf("%.4f", s.longitude))
	query.Set("current", "temperature_2m,apparent_temperature,relative_humidity_2m,precipitation,rain,snowfall,weather_code,wind_speed_10m,wind_gusts_10m")
	query.Set("wind_speed_unit", "ms")
	query.Set("timezone", "auto")
	endpoint.RawQuery = query.Encode()

	var response openMeteoResponse
	if err := getJSON(ctx, s.client, endpoint.String(), &response); err != nil {
		return model.WeatherCurrent{}, fmt.Errorf("get open-meteo forecast: %w", err)
	}

	current := response.Current
	condition := conditionFromWeather(current.WeatherCode, current.RainMM, current.SnowfallCM)

	weather := model.WeatherCurrent{
		City:                 s.city,
		Condition:            condition,
		TemperatureC:         current.TemperatureC,
		ApparentTemperatureC: current.ApparentTemperatureC,
		Rain:                 current.RainMM > 0 || current.PrecipitationMM > 0,
		Snow:                 current.SnowfallCM > 0,
		Humidity:             current.Humidity,
		PrecipitationMM:      current.PrecipitationMM,
		WeatherCode:          current.WeatherCode,
		WindSpeedMS:          current.WindSpeedMS,
		WindGustsMS:          current.WindGustsMS,
		RoadSalt:             current.SnowfallCM > 0 || current.ApparentTemperatureC <= 0 && current.PrecipitationMM > 0,
		Source:               "open-meteo",
	}
	weather.RideAdvice = rideAdviceFromWeather(weather)

	return weather, nil
}

func getJSON(ctx context.Context, client *http.Client, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func conditionFromWeather(code int, rainMM float64, snowfallCM float64) string {
	if snowfallCM > 0 || code >= 71 && code <= 77 || code >= 85 && code <= 86 {
		return "snowy"
	}

	if code >= 95 {
		return "stormy"
	}

	if rainMM > 0 || code >= 51 && code <= 67 || code >= 80 && code <= 82 {
		return "rainy"
	}

	if code >= 45 && code <= 48 {
		return "foggy"
	}

	return "dry"
}

func (s *weatherService) fallbackWeather() model.WeatherCurrent {
	weather := model.WeatherCurrent{
		City:                 s.city,
		Condition:            "rainy",
		TemperatureC:         12.0,
		ApparentTemperatureC: 9.0,
		Rain:                 true,
		Snow:                 false,
		Humidity:             76,
		PrecipitationMM:      1.2,
		WeatherCode:          61,
		WindSpeedMS:          5.5,
		WindGustsMS:          9.0,
		RoadSalt:             false,
		Source:               "fallback",
	}
	weather.RideAdvice = rideAdviceFromWeather(weather)
	return weather
}

func riskFromWeather(weather model.WeatherCurrent) model.WeatherRisk {
	risk := model.WeatherRisk{
		City:                 weather.City,
		Condition:            weather.Condition,
		Source:               weather.Source,
		TemperatureC:         weather.TemperatureC,
		ApparentTemperatureC: weather.ApparentTemperatureC,
		PrecipitationMM:      weather.PrecipitationMM,
		WindSpeedMS:          weather.WindSpeedMS,
		WindGustsMS:          weather.WindGustsMS,
		RideAdvice:           weather.RideAdvice,
	}

	if weather.Snow || weather.RoadSalt || weather.Condition == "stormy" || weather.PrecipitationMM >= 4 {
		risk.Risk = "high"
		risk.Reason = "snow, storms, road salt or heavy rain increase corrosion and drivetrain wear"
		return risk
	}

	if weather.Rain || weather.PrecipitationMM > 0 || weather.Humidity >= 75 {
		risk.Risk = "medium"
		risk.Reason = "wet or humid conditions increase the need for chain lubrication"
		return risk
	}

	risk.Risk = "low"
	risk.Reason = "dry weather keeps the normal maintenance interval"
	return risk
}

func rideAdviceFromWeather(weather model.WeatherCurrent) model.RideAdvice {
	gear := make([]string, 0, 5)
	afterRide := make([]string, 0, 3)

	if weather.ApparentTemperatureC <= 3 {
		gear = append(gear, "warm gloves")
	} else if weather.ApparentTemperatureC <= 10 || weather.WindSpeedMS >= 7 {
		gear = append(gear, "full-finger gloves")
	}

	if weather.WindSpeedMS >= 7 || weather.WindGustsMS >= 10 {
		gear = append(gear, "windproof layer")
	}

	if weather.Rain || weather.Snow || weather.PrecipitationMM > 0 {
		gear = append(gear, "waterproof jacket")
		gear = append(gear, "bike lights")
		afterRide = append(afterRide, "wipe the drivetrain after the ride")
		afterRide = append(afterRide, "lubricate the chain if it got wet")
	}

	if weather.Condition == "foggy" || weather.Condition == "stormy" {
		gear = append(gear, "bike lights")
	}

	if weather.RoadSalt {
		afterRide = append(afterRide, "rinse road salt from the drivetrain")
	}

	if len(gear) == 0 {
		gear = append(gear, "standard helmet")
	}

	if weather.Condition == "stormy" || weather.Snow || weather.WindGustsMS >= 17 || weather.WindSpeedMS >= 14 || weather.ApparentTemperatureC <= -5 {
		return model.RideAdvice{
			Status:    "not_recommended",
			Title:     "Ride not recommended",
			Message:   "Current weather can make handling, braking or visibility unsafe. Consider postponing the ride.",
			CanRide:   false,
			Gear:      uniqueStrings(gear),
			AfterRide: uniqueStrings(afterRide),
		}
	}

	if weather.Rain || weather.PrecipitationMM > 0 || weather.WindGustsMS >= 10 || weather.WindSpeedMS >= 9 || weather.ApparentTemperatureC <= 8 || weather.Condition == "foggy" {
		return model.RideAdvice{
			Status:    "caution",
			Title:     "Ride with caution",
			Message:   "Weather can affect braking distance, grip and bike handling. Slow down and choose a safer route.",
			CanRide:   true,
			Gear:      uniqueStrings(gear),
			AfterRide: uniqueStrings(afterRide),
		}
	}

	return model.RideAdvice{
		Status:    "good",
		Title:     "Good riding conditions",
		Message:   "Weather looks suitable for a normal ride. Do a quick tire, brake and chain check before leaving.",
		CanRide:   true,
		Gear:      uniqueStrings(gear),
		AfterRide: uniqueStrings(afterRide),
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func firstNonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
