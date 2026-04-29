package config

import (
	"os"
	"strconv"
	"time"
)

const defaultHTTPTimeout = 3 * time.Second

type Config struct {
	ServiceName string
	Port        string

	HTTPTimeout  time.Duration
	DemoWeather  string
	DemoLatency  time.Duration
	DemoFailRate float64

	WeatherCity      string
	WeatherLatitude  float64
	WeatherLongitude float64

	BikeAPIURL        string
	MaintenanceAPIURL string
	WeatherAPIURL     string
	ReminderAPIURL    string
	UserAPIURL        string
}

func Load(serviceName string) Config {
	return Config{
		ServiceName:       env("SERVICE_NAME", serviceName),
		Port:              env("PORT", defaultPort(serviceName)),
		HTTPTimeout:       envDuration("HTTP_TIMEOUT", defaultHTTPTimeout),
		DemoWeather:       env("DEMO_WEATHER", "rainy"),
		DemoLatency:       time.Duration(envInt("DEMO_LATENCY_MS", 0)) * time.Millisecond,
		DemoFailRate:      clamp01(envFloat("DEMO_FAIL_RATE", 0)),
		WeatherCity:       env("WEATHER_CITY", "Vienna"),
		WeatherLatitude:   envFloat("WEATHER_LATITUDE", 48.2082),
		WeatherLongitude:  envFloat("WEATHER_LONGITUDE", 16.3738),
		BikeAPIURL:        env("BIKE_API_URL", "http://localhost:8081"),
		MaintenanceAPIURL: env("MAINTENANCE_API_URL", "http://localhost:8082"),
		WeatherAPIURL:     env("WEATHER_API_URL", "http://localhost:8083"),
		ReminderAPIURL:    env("REMINDER_API_URL", "http://localhost:8084"),
		UserAPIURL:        env("USER_API_URL", "http://localhost:8085"),
	}
}

func (c Config) Addr() string {
	return ":" + c.Port
}

func defaultPort(serviceName string) string {
	switch serviceName {
	case "frontend":
		return "8080"
	case "bike-api":
		return "8081"
	case "maintenance-api":
		return "8082"
	case "weather-api":
		return "8083"
	case "reminder-api":
		return "8084"
	case "user-api":
		return "8085"
	default:
		return "8080"
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
