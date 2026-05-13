# Project Scope and Definition of Done

## Project goal

Build a production-like SRE lab for a Kubernetes-based microservice application.
The project demonstrates the full operational lifecycle: deploying Chainwise to Kubernetes, collecting service metrics, defining SLI/SLO, calculating error budget, configuring burn-rate alerts, routing alerts, writing runbooks, running reproducible incidents, and documenting postmortems.

## Approved assignment stages

1. Define project scope, assumptions, deliverables and Definition of Done.
2. Build a demo microservice application with health, readiness and metrics endpoints.
3. Containerize the application and publish images to a container registry.
4. Deploy Chainwise to a Kubernetes cluster.
5. Add Kubernetes health checks and basic resource configuration.
6. Install the observability stack: Prometheus, Alertmanager and Grafana.
7. Configure Prometheus scraping for Chainwise services.
8. Build Grafana dashboards for service and SLO overview.
9. Define SLI/SLO and error budget.
10. Configure recording rules and burn-rate alerts.
11. Configure Alertmanager routing, grouping, silences and inhibition.
12. Write runbooks for key alerts.
13. Run reproducible incident scenarios and collect evidence.
14. Write blameless postmortems.
15. Prepare README, demo script and final report.

## In scope

- Local or self-managed Kubernetes cluster.
- Chainwise demo microservice application.
- Docker images published to Docker Hub.
- Kubernetes Deployments and Services.
- Liveness and readiness probes.
- CPU and memory requests/limits.
- Prometheus-based metrics collection.
- Grafana dashboards.
- SLI/SLO model for availability and latency.
- Error budget calculation.
- Burn-rate alerting.
- Alertmanager routing and noise-control examples.
- Runbooks for key alerts.
- Reproducible incident scenarios.
- Evidence files and screenshots.
- Final project report.

## Out of scope

- User authentication and authorization.
- Real user accounts.
- Persistent application database.
- Real notification delivery to external users.
- Payment logic.
- Production-grade frontend design.
- Multi-region deployment.
- Cloud-managed Kubernetes.
- Production secrets management.
- Real customer data.

## Assumptions

- The project runs in a local or self-managed Kubernetes environment.
- Docker Hub is used as the container registry for Chainwise images.
- Chainwise is a demo application and does not store real user data.
- Open-Meteo may be used as an external weather data source.
- The main goal is SRE methodology, not business application complexity.
- Observability components will be installed in a separate namespace.
- Application components will run in the `chainwise` namespace.

## Expected deliverables

- Source code for Chainwise microservices.
- Dockerfile and Docker Compose configuration.
- Kubernetes manifests for Chainwise.
- Prometheus/Grafana/Alertmanager configuration.
- SLI/SLO documentation.
- Prometheus recording and alerting rules.
- Runbooks.
- Incident evidence.
- Postmortems.
- README with setup and demo instructions.
- Final report.

## Definition of Done

The project is considered complete when:

1. Chainwise can be deployed reproducibly to Kubernetes.
2. All Chainwise pods are Running and Ready.
3. Prometheus collects metrics from Chainwise services.
4. Grafana contains service and SLO dashboards.
5. At least two SLOs are defined and documented.
6. Error budget is calculated for the selected SLOs.
7. Burn-rate alerts are configured for SLO violations.
8. Alertmanager routes alerts by severity.
9. At least one silence and one inhibition example are demonstrated.
10. Each key alert has a runbook.
11. At least two incident scenarios are reproducible.
12. Evidence is saved for incidents and validation steps.
13. At least two blameless postmortems are written.
14. README contains setup, validation and demo instructions.
15. The final report explains the methodology, implementation, experiments and conclusions.
