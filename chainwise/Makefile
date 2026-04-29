SERVICES := frontend bike-api maintenance-api weather-api reminder-api user-api

.PHONY: fmt tidy test build clean run-frontend run-bike-api run-maintenance-api run-weather-api run-reminder-api run-user-api docker-build

fmt:
	go fmt ./...

tidy:
	go mod tidy

test:
	go test ./...

build:
	mkdir -p bin
	for service in $(SERVICES); do \
		go build -o bin/$$service ./cmd/$$service; \
	done

clean:
	rm -rf bin

run-frontend:
	PORT=8080 BIKE_API_URL=http://localhost:8081 go run ./cmd/frontend

run-bike-api:
	PORT=8081 MAINTENANCE_API_URL=http://localhost:8082 go run ./cmd/bike-api

run-maintenance-api:
	PORT=8082 WEATHER_API_URL=http://localhost:8083 go run ./cmd/maintenance-api

run-weather-api:
	PORT=8083 REMINDER_API_URL=http://localhost:8084 go run ./cmd/weather-api

run-reminder-api:
	PORT=8084 USER_API_URL=http://localhost:8085 go run ./cmd/reminder-api

run-user-api:
	PORT=8085 go run ./cmd/user-api

docker-build:
	for service in $(SERVICES); do \
		docker build --build-arg SERVICE=$$service -t chainwise/$$service:local .; \
	done
