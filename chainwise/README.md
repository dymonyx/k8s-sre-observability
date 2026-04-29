# Chainwise

**Chainwise** is a Go-based microservice application for bicycle maintenance recommendations.

The application helps cyclists understand when they should clean, inspect, lubricate, or replace bicycle components based on bicycle usage, weather conditions, and user preferences. It is designed as a simple but realistic distributed application that can later be deployed to Kubernetes and used for observability, reliability, and service mesh experiments.

---

## Application Idea

Cyclists often forget regular maintenance tasks such as lubricating the chain, checking tire pressure, inspecting brake pads, cleaning the drivetrain, or replacing worn components. Chainwise provides maintenance recommendations by collecting information about a bike, evaluating current conditions, and generating a suggested maintenance plan.

A typical user flow:

1. The user opens the frontend.
2. The user requests a maintenance check for their bicycle.
3. The application checks the bicycle profile and usage data.
4. The application evaluates weather-related maintenance risks.
5. The application generates a maintenance recommendation.
6. The application creates a reminder for the next maintenance action.
7. The user receives a clear recommendation and next service date.

---

## Service Architecture

The planned service chain is:

```text
frontend → bike-api → maintenance-api → weather-api → reminder-api → user-api
```

Each service is planned to be implemented in **Go** as a small HTTP API.

---

## Services

### `frontend`

The user-facing entry point of the application.

Responsibilities:

* Provide a simple web page or HTTP interface.
* Allow users to request a bicycle maintenance check.
* Send requests to `bike-api`.
* Display the final recommendation returned by the backend services.

Example features:

* Maintenance check button.
* Display of bike profile.
* Display of recommended maintenance actions.
* Display of next reminder date.

---

### `bike-api`

The main backend entry point for bicycle-related operations.

Responsibilities:

* Receive maintenance check requests from the frontend.
* Validate bicycle information.
* Prepare a bicycle profile for recommendation calculation.
* Call `maintenance-api` to calculate the final maintenance advice.
* Return the aggregated result to the frontend.

Example bicycle profile fields:

```text
bike type
last service date
weekly distance
riding style
chain condition
brake condition
tire condition
```

---

### `maintenance-api`

The core business service of Chainwise.

Responsibilities:

* Calculate maintenance recommendations.
* Decide which bike components need attention.
* Adjust recommendations based on usage and weather data.
* Call `weather-api` to include environmental conditions.
* Pass reminder information further down the chain.

Example recommendations:

```text
Clean and lubricate the chain.
Check tire pressure this week.
Inspect brake pads after the next 100 km.
Clean the drivetrain after wet weather rides.
Schedule a full inspection in two weeks.
```

---

### `weather-api`

A weather context service used by the maintenance logic.

Responsibilities:

* Provide simplified weather information.
* Estimate how weather affects bicycle maintenance needs.
* Return risk factors such as rain, snow, humidity, or road salt.
* Call `reminder-api` to help schedule weather-aware reminders.

Example weather impact rules:

```text
Rain increases the need for chain lubrication.
Snow and road salt increase the need for drivetrain cleaning.
High humidity increases corrosion risk.
Dry weather keeps the normal maintenance interval.
```

---

### `reminder-api`

A service for creating and managing maintenance reminders.

Responsibilities:

* Generate the next recommended maintenance date.
* Create a reminder for the user.
* Adjust reminder urgency based on maintenance priority.
* Call `user-api` to get user preferences and notification settings.

Example reminder types:

```text
chain lubrication reminder
brake inspection reminder
tire pressure reminder
drivetrain cleaning reminder
full service reminder
```

---

### `user-api`

A user profile and preferences service.

Responsibilities:

* Store or return demo user information.
* Provide notification preferences.
* Provide cycling habits and maintenance preferences.
* Return user-specific settings to `reminder-api`.

Example user preferences:

```text
preferred reminder frequency
notification channel
average weekly riding distance
maintenance experience level
bike usage type
```

---

## Planned Go Implementation

Each service will be implemented as a small Go HTTP server.

Planned common endpoints for every service:

```text
GET /healthz     Liveness check
GET /readyz      Readiness check
GET /metrics     Prometheus-compatible metrics endpoint
```

Planned service-specific endpoints:

```text
frontend:
  GET /
  GET /check

bike-api:
  GET /bike/check
  GET /bike/profile

maintenance-api:
  GET /maintenance/recommendation

weather-api:
  GET /weather/current
  GET /weather/risk

reminder-api:
  GET /reminders/next
  POST /reminders

user-api:
  GET /users/demo
  GET /users/preferences
```

---

## Main Features

### Bicycle Maintenance Recommendations

Chainwise generates practical maintenance advice based on bicycle usage and condition.

Example output:

```json
{
  "bike": "gravel bike",
  "recommendation": "Clean and lubricate the chain",
  "priority": "medium",
  "reason": "Recent wet weather increases drivetrain wear",
  "nextReminder": "2026-05-06"
}
```

---

### Weather-Aware Maintenance Logic

Weather conditions influence the maintenance recommendation. For example, riding in rain or snow can shorten the recommended chain lubrication interval.

---

### Reminder Generation

The application can suggest the next maintenance date and create a reminder based on the recommendation priority and user preferences.

---

### User Preferences

The application can adapt recommendations to user settings such as riding frequency, bike type, preferred reminder interval, and maintenance experience level.

---

### Demo Failure Modes

For future reliability testing, some services may include optional demo failure modes:

```text
slow response
random error
temporary dependency failure
```

These modes can be useful for testing monitoring, alerting, and incident response later.

---

## Example Full Request Flow

```text
1. User opens the frontend.
2. frontend calls bike-api.
3. bike-api loads or creates a bicycle profile.
4. bike-api calls maintenance-api.
5. maintenance-api asks weather-api for current maintenance risk.
6. weather-api calls reminder-api to create a weather-aware reminder.
7. reminder-api calls user-api to get user preferences.
8. The final result is returned back through the chain.
9. The user receives a maintenance recommendation.
```

---

## Example Use Case

A user owns a gravel bike and rides around 80 km per week. Recently, the weather has been rainy. Chainwise detects that wet rides increase drivetrain wear and recommends cleaning and lubricating the chain earlier than usual.

Example result:

```text
Recommendation: Clean and lubricate the chain
Priority: Medium
Reason: Rainy weather increases drivetrain wear
Next reminder: In 5 days
```

---

## Project Purpose

The first goal of Chainwise is to provide a useful and understandable Go microservice application.

Later, the same application can be used as a demo workload for:

```text
Kubernetes deployments
Istio Service Mesh
service-to-service communication
observability
SLI/SLO monitoring
alerting
incident response scenarios
```

---

## Status

Current status:

```text
Application concept: ready
Service architecture: ready
Implementation language: Go
Next step: implement minimal Go services
```


----------------------------------
# Chainwise

Chainwise is a small Go microservice demo application for bicycle maintenance recommendations. It is designed to be deployed to Kubernetes and used for SRE, observability, SLI/SLO, alerting, service mesh and incident response practice.

## Services

```text
frontend -> bike-api -> maintenance-api -> weather-api -> reminder-api -> user-api
```

Every service exposes:

```text
GET /healthz
GET /readyz
GET /metrics
```

Main application endpoints:

```text
frontend:        GET /, GET /check
bike-api:        GET /bike/check, GET /bike/profile
maintenance-api: GET /maintenance/recommendation
weather-api:     GET /weather/current, GET /weather/risk
reminder-api:    GET /reminders/next, POST /reminders
user-api:        GET /users/demo, GET /users/preferences
```

## Local development

Install dependencies:

```bash
go mod tidy
```

Run services in separate terminals:

```bash
make run-user-api
make run-reminder-api
make run-weather-api
make run-maintenance-api
make run-bike-api
make run-frontend
```

Open:

```text
http://localhost:8080
http://localhost:8080/check
```

Or run with Docker Compose:

```bash
docker compose up --build
```

## Demo failure modes

Each service supports optional env vars for SRE experiments:

```bash
DEMO_LATENCY_MS=500
DEMO_FAIL_RATE=0.2
```

For weather-aware recommendations:

```bash
DEMO_WEATHER=rainy    # dry, rainy, snowy, humid
```
