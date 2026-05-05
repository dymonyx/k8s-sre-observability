# k8s-sre-observability

A Kubernetes-based SRE observability project focused on SLI/SLO monitoring, alerting, incident response, and reliability practices.

The project uses **Chainwise**, a demo Go microservice application, as a target service for building and validating an SRE monitoring methodology in Kubernetes.

## Project Topic

**Development of a methodology for monitoring and incident response in a Kubernetes cluster based on SLI/SLO.**

The main goal is not only to run an application, but to demonstrate a complete SRE workflow around it:

metrics → SLI/SLO → error budget → burn-rate alerts → Alertmanager → runbooks → incident response → postmortems

## Tech Stack

* **Go** — demo microservice application
* **Docker / Docker Compose** — local development and testing
* **Kubernetes** — deployment environment
* **Helm** — monitoring stack installation
* **Prometheus** — metrics collection and alerting rules
* **Grafana** — dashboards and visualization
* **Alertmanager** — alert routing, grouping, silencing, and inhibition
* **k6** — load testing and incident simulation
* **SLI/SLO** — reliability model based on user-facing symptoms

## Demo Application: Chainwise

Chainwise is a bicycle maintenance recommendation application built as a set of Go microservices.

The application consists of:

* frontend
* bike-api
* maintenance-api
* weather-api
* reminder-api
* user-api

The user opens the frontend, enters bike usage data, and receives a maintenance recommendation based on odometer values, component condition, and weather risk.

Request flow:

Browser
↓
frontend
↓
bike-api
↓
maintenance-api
↓
weather-api
↓
reminder-api
↓
user-api

## Current Application Features

The demo application includes:

* HTTP microservice architecture
* health endpoints: `/healthz`
* readiness endpoints: `/readyz`
* Prometheus-compatible metrics: `/metrics`
* structured JSON logging
* request ID propagation between services
* configurable fault injection
* Docker Compose setup for local development

Fault injection supports:

* artificial latency
* artificial failures
* reproducible incident scenarios

## Repository Structure

* `chainwise/` — demo Go microservice application
* `chainwise/cmd/` — service entrypoints
* `chainwise/internal/` — shared internal packages
* `chainwise/docker-compose.yml` — local development setup
* `chainwise/Dockerfile` — service image build
* `docs/` — project documentation
* `k8s/` — Kubernetes manifests
* `monitoring/` — Prometheus, Grafana, Alertmanager configs
* `runbooks/` — incident response runbooks
* `load/` — k6 load test scripts
* `reports/` — final report and postmortems

Some directories may be added or completed during the next project stages.

## Local Development

To run Chainwise locally with Docker Compose:

1. Open the project directory.
2. Go to the Chainwise application folder.
3. Build and start all services with Docker Compose.

Commands:

`cd chainwise`

`docker compose build`

`docker compose up`

Frontend will be available at:

`http://localhost:8080`

Main service ports:

* frontend — `8080`
* bike-api — `8081`
* maintenance-api — `8082`
* weather-api — `8083`
* reminder-api — `8084`
* user-api — `8085`

Example checks:

`curl http://localhost:8080/healthz`

`curl http://localhost:8080/readyz`

`curl http://localhost:8080/metrics`

## Fault Injection

The application supports artificial degradation for incident simulation.

Environment variables:

* `DEMO_LATENCY_MS=1000` — adds artificial response delay
* `DEMO_FAIL_RATE=0.2` — makes part of requests fail

These variables can be used to simulate:

* high latency incidents
* increased 5xx error rate
* SLO degradation
* burn-rate alert triggering

## Kubernetes Deployment

The project is intended to run in a Kubernetes cluster.

Target environment:

* 3-node Kubernetes cluster

Planned Kubernetes resources:

* Namespace
* Deployments
* Services
* liveness probes
* readiness probes
* resource requests and limits
* ServiceMonitor resources for Prometheus scraping

Example deployment flow:

`kubectl apply -f k8s/`

`kubectl get pods -n chainwise`

`kubectl get svc -n chainwise`

Frontend access example:

`kubectl -n chainwise port-forward svc/frontend 8080:8080`

## Observability Plan

The monitoring stack will be based on **kube-prometheus-stack**.

It provides:

* Prometheus
* Grafana
* Alertmanager
* Prometheus Operator
* Kubernetes monitoring components

Planned monitoring flow:

Chainwise `/metrics`
↓
ServiceMonitor
↓
Prometheus
↓
PrometheusRule / recording rules
↓
Alertmanager
↓
Runbooks and incident response

## SLI/SLO Model

The project will define at least two user-facing SLOs.

### Availability SLO

Measures the percentage of successful requests for the main user flow.

Example:

99% of requests to the recommendation flow should not return 5xx errors.

### Latency SLO

Measures how fast the main user flow responds.

Example:

95% of recommendation requests should complete under 500 ms.

These SLOs will be used to calculate:

* error budget
* burn rate
* alerting conditions

## Alerting and Incident Response

The project will include SLO-based alerting using Prometheus and Alertmanager.

Planned alerting features:

* availability burn-rate alerts
* latency burn-rate alerts
* page-level alerts
* ticket-level alerts
* Alertmanager routing
* alert grouping
* silences
* inhibition rules
* runbook links in alert annotations

## Runbooks

Runbooks will describe how to investigate and mitigate incidents.

Planned runbooks:

* High 5xx error rate
* High latency
* Kubernetes pod not ready

Each runbook will include:

* meaning
* user impact
* quick triage
* mitigation steps
* verification
* follow-up actions

## Incident Scenarios

The methodology will be validated through reproducible incidents.

### Scenario 1: High error rate

Flow:

k6 traffic → enable `DEMO_FAIL_RATE` → Prometheus detects errors → burn-rate alert fires → mitigation → alert resolves

### Scenario 2: High latency

Flow:

k6 traffic → enable `DEMO_LATENCY_MS` → Prometheus detects latency degradation → alert fires → mitigation → alert resolves

For each scenario, the project will measure:

* MTTD — Mean Time To Detect
* MTTR — Mean Time To Resolve

## Project Stages

The approved project plan contains seven stages:

1. Requirements clarification and formalization
2. Domain analysis and monitoring methodology selection
3. Demo service development
4. Monitoring and response architecture design
5. Infrastructure and monitoring system deployment
6. SLI/SLO-based incident response implementation
7. Methodology testing and final report writing

## Out of Scope

The current demo version does not include:

* user authentication
* database integration
* real email or push notifications
* production-grade security hardening
* cloud provider infrastructure
* real on-call integration

These features are intentionally excluded because the project focuses on SRE monitoring and incident response methodology.

## Definition of Done

The project is considered complete when:

* Chainwise is deployed in Kubernetes
* Prometheus collects metrics from Chainwise services
* Grafana dashboards show service health and SLO status
* availability and latency SLOs are defined
* error budget and burn-rate alerts are implemented
* Alertmanager routes alerts correctly
* runbooks exist for key alerts
* at least two incident scenarios are reproduced
* MTTD and MTTR are measured
* postmortems are written
* final report and demo script are prepared

## Project Status

Current progress:

* Demo application: done
* Docker Compose setup: done
* Health/readiness/metrics endpoints: done
* Fault injection: done
* 3-node Kubernetes cluster: done
* Kubernetes application deployment: in progress
* Monitoring stack: planned
* SLI/SLO alerting: planned
* Incident scenarios: planned
* Final report: planned

## License

This project is licensed under the Apache-2.0 License.
