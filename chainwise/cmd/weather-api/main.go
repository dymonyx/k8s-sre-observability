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
		TemperatureC    float64 `json:"temperature_2m"`
		Humidity        int     `json:"relative_humidity_2m"`
		PrecipitationMM float64 `json:"precipitation"`
		RainMM          float64 `json:"rain"`
		SnowfallCM      float64 `json:"snowfall"`
		WeatherCode     int     `json:"weather_code"`
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
	query.Set("current", "temperature_2m,relative_humidity_2m,precipitation,rain,snowfall,weather_code")
	query.Set("timezone", "auto")
	endpoint.RawQuery = query.Encode()

	var response openMeteoResponse
	if err := getJSON(ctx, s.client, endpoint.String(), &response); err != nil {
		return model.WeatherCurrent{}, fmt.Errorf("get open-meteo forecast: %w", err)
	}

	current := response.Current
	condition := conditionFromWeather(current.WeatherCode, current.RainMM, current.SnowfallCM)

	return model.WeatherCurrent{
		City:            s.city,
		Condition:       condition,
		TemperatureC:    current.TemperatureC,
		Rain:            current.RainMM > 0 || current.PrecipitationMM > 0,
		Snow:            current.SnowfallCM > 0,
		Humidity:        current.Humidity,
		PrecipitationMM: current.PrecipitationMM,
		WeatherCode:     current.WeatherCode,
		RoadSalt:        current.SnowfallCM > 0,
		Source:          "open-meteo",
	}, nil
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

	if rainMM > 0 || code >= 51 && code <= 67 || code >= 80 && code <= 82 {
		return "rainy"
	}

	if code >= 95 {
		return "stormy"
	}

	if code >= 45 && code <= 48 {
		return "foggy"
	}

	return "dry"
}

func (s *weatherService) fallbackWeather() model.WeatherCurrent {
	return model.WeatherCurrent{
		City:            s.city,
		Condition:       "rainy",
		TemperatureC:    12.0,
		Rain:            true,
		Snow:            false,
		Humidity:        76,
		PrecipitationMM: 1.2,
		WeatherCode:     61,
		RoadSalt:        false,
		Source:          "fallback",
	}
}

func riskFromWeather(weather model.WeatherCurrent) model.WeatherRisk {
	if weather.Snow || weather.RoadSalt || weather.Condition == "stormy" {
		return model.WeatherRisk{
			City:      weather.City,
			Condition: weather.Condition,
			Risk:      "high",
			Reason:    "snow, storms or road salt increase corrosion and drivetrain wear",
			Source:    weather.Source,
		}
	}

	if weather.Rain || weather.PrecipitationMM > 0 || weather.Humidity >= 75 {
		return model.WeatherRisk{
			City:      weather.City,
			Condition: weather.Condition,
			Risk:      "medium",
			Reason:    "wet conditions increase the need for chain lubrication",
			Source:    weather.Source,
		}
	}

	return model.WeatherRisk{
		City:      weather.City,
		Condition: weather.Condition,
		Risk:      "low",
		Reason:    "dry weather keeps the normal maintenance interval",
		Source:    weather.Source,
	}
}

func firstNonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
