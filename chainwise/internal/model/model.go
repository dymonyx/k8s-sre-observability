package model

type HealthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

type BikeProfile struct {
	ID                             string  `json:"id"`
	Name                           string  `json:"name"`
	Type                           string  `json:"type"`
	CurrentOdometerKM              int     `json:"currentOdometerKm"`
	LastRideDistanceKM             int     `json:"lastRideDistanceKm"`
	LastRideDate                   string  `json:"lastRideDate"`
	LastServiceDate                string  `json:"lastServiceDate"`
	LastServiceOdometerKM          int     `json:"lastServiceOdometerKm"`
	LastChainLubeOdometerKM        int     `json:"lastChainLubeOdometerKm"`
	LastChainReplacementOdometerKM int     `json:"lastChainReplacementOdometerKm"`
	LastBrakeCheckOdometerKM       int     `json:"lastBrakeCheckOdometerKm"`
	LastTireCheckOdometerKM        int     `json:"lastTireCheckOdometerKm"`
	RidingStyle                    string  `json:"ridingStyle"`
	ChainCondition                 string  `json:"chainCondition"`
	ChainWear                      string  `json:"chainWear"`
	BrakeCondition                 string  `json:"brakeCondition"`
	BrakePadThickness              string  `json:"brakePadThickness"`
	BrakeSymptoms                  string  `json:"brakeSymptoms"`
	TireCondition                  string  `json:"tireCondition"`
	RecentPunctures                int     `json:"recentPunctures"`
	FrontTirePressureBar           float64 `json:"frontTirePressureBar"`
	RearTirePressureBar            float64 `json:"rearTirePressureBar"`
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

type RideAdvice struct {
	Status    string   `json:"status"`
	Title     string   `json:"title"`
	Message   string   `json:"message"`
	CanRide   bool     `json:"canRide"`
	Gear      []string `json:"gear"`
	AfterRide []string `json:"afterRide"`
}

type WeatherCurrent struct {
	City                 string     `json:"city"`
	Condition            string     `json:"condition"`
	TemperatureC         float64    `json:"temperatureC"`
	ApparentTemperatureC float64    `json:"apparentTemperatureC"`
	Rain                 bool       `json:"rain"`
	Snow                 bool       `json:"snow"`
	Humidity             int        `json:"humidity"`
	PrecipitationMM      float64    `json:"precipitationMm"`
	WeatherCode          int        `json:"weatherCode"`
	WindSpeedMS          float64    `json:"windSpeedMs"`
	WindGustsMS          float64    `json:"windGustsMs"`
	RoadSalt             bool       `json:"roadSalt"`
	Source               string     `json:"source"`
	RideAdvice           RideAdvice `json:"rideAdvice"`
}

type WeatherRisk struct {
	City                 string     `json:"city"`
	Condition            string     `json:"condition"`
	Risk                 string     `json:"risk"`
	Reason               string     `json:"reason"`
	Source               string     `json:"source"`
	TemperatureC         float64    `json:"temperatureC"`
	ApparentTemperatureC float64    `json:"apparentTemperatureC"`
	PrecipitationMM      float64    `json:"precipitationMm"`
	WindSpeedMS          float64    `json:"windSpeedMs"`
	WindGustsMS          float64    `json:"windGustsMs"`
	RideAdvice           RideAdvice `json:"rideAdvice"`
	Reminder             *Reminder  `json:"reminder,omitempty"`
}

type ComponentForecast struct {
	Component   string `json:"component"`
	Label       string `json:"label"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	KmSince     int    `json:"kmSince"`
	IntervalKM  int    `json:"intervalKm"`
	RemainingKM int    `json:"remainingKm"`
	OverdueKM   int    `json:"overdueKm"`
	Action      string `json:"action"`
	Reason      string `json:"reason"`
}

type MaintenanceRecommendation struct {
	Bike              string              `json:"bike"`
	Recommendation    string              `json:"recommendation"`
	Priority          string              `json:"priority"`
	Reason            string              `json:"reason"`
	KmSinceService    int                 `json:"kmSinceService"`
	KmSinceChainLube  int                 `json:"kmSinceChainLube"`
	WeatherRisk       WeatherRisk         `json:"weatherRisk"`
	ComponentForecast []ComponentForecast `json:"componentForecast"`
	NextReminder      string              `json:"nextReminder"`
}

type BikeCheckResponse struct {
	BikeProfile    BikeProfile               `json:"bikeProfile"`
	Recommendation MaintenanceRecommendation `json:"recommendation"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
