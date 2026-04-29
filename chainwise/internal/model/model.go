package model

type HealthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

type BikeProfile struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Type                    string `json:"type"`
	CurrentOdometerKM       int    `json:"currentOdometerKm"`
	LastRideDistanceKM      int    `json:"lastRideDistanceKm"`
	LastRideDate            string `json:"lastRideDate"`
	LastServiceDate         string `json:"lastServiceDate"`
	LastServiceOdometerKM   int    `json:"lastServiceOdometerKm"`
	LastChainLubeOdometerKM int    `json:"lastChainLubeOdometerKm"`
	RidingStyle             string `json:"ridingStyle"`
	ChainCondition          string `json:"chainCondition"`
	BrakeCondition          string `json:"brakeCondition"`
	TireCondition           string `json:"tireCondition"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level string `json:"level"`
}

type UserPreferences struct {
	UserID                  string `json:"userId"`
	PreferredReminderEvery  string `json:"preferredReminderEvery"`
	NotificationChannel     string `json:"notificationChannel"`
	AverageWeeklyDistanceKM int    `json:"averageWeeklyDistanceKm"`
	MaintenanceExperience   string `json:"maintenanceExperience"`
	BikeUsageType           string `json:"bikeUsageType"`
}

type Reminder struct {
	Type     string `json:"type"`
	Priority string `json:"priority"`
	NextDate string `json:"nextDate"`
	Channel  string `json:"channel"`
	Message  string `json:"message"`
}

type WeatherCurrent struct {
	City            string  `json:"city"`
	Condition       string  `json:"condition"`
	TemperatureC    float64 `json:"temperatureC"`
	Rain            bool    `json:"rain"`
	Snow            bool    `json:"snow"`
	Humidity        int     `json:"humidity"`
	PrecipitationMM float64 `json:"precipitationMm"`
	WeatherCode     int     `json:"weatherCode"`
	RoadSalt        bool    `json:"roadSalt"`
	Source          string  `json:"source"`
}

type WeatherRisk struct {
	City      string    `json:"city"`
	Condition string    `json:"condition"`
	Risk      string    `json:"risk"`
	Reason    string    `json:"reason"`
	Source    string    `json:"source"`
	Reminder  *Reminder `json:"reminder,omitempty"`
}

type MaintenanceRecommendation struct {
	Bike             string      `json:"bike"`
	Recommendation   string      `json:"recommendation"`
	Priority         string      `json:"priority"`
	Reason           string      `json:"reason"`
	KmSinceService   int         `json:"kmSinceService"`
	KmSinceChainLube int         `json:"kmSinceChainLube"`
	WeatherRisk      WeatherRisk `json:"weatherRisk"`
	NextReminder     string      `json:"nextReminder"`
}

type BikeCheckResponse struct {
	BikeProfile    BikeProfile               `json:"bikeProfile"`
	Recommendation MaintenanceRecommendation `json:"recommendation"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
