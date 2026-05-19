# Runbook: Chainwise Availability Burn Rate

## Meaning

This runbook is used when the Chainwise availability SLO for the main recommendation flow is being violated or the error budget is being consumed too quickly.

Related alerts:

```text
ChainwiseAvailabilityBurnRatePage
ChainwiseAvailabilityBurnRateTicket
```

The affected user-facing endpoint is:

```text
frontend /check
```

This endpoint represents the main Chainwise recommendation flow:

```text
frontend /check
  -> bike-api /bike/check
    -> maintenance-api /maintenance/recommendation
      -> weather-api /weather/risk
        -> reminder-api /reminders/next
          -> user-api /users/preferences
```

The availability SLO target is:

```text
99.5% of frontend /check requests should not return 5xx errors over a rolling 1-day window.
```

The availability error budget is:

```text
0.5%
```

The burn rate shows how quickly this error budget is being consumed.

## Impact

Users may fail to receive bicycle maintenance recommendations.

Possible user-visible symptoms:

- `/check` returns 5xx errors;
- the Chainwise UI cannot display a recommendation;
- dependent backend services may be unavailable or failing;
- the error budget is consumed faster than allowed.

## Quick Triage

Check the current SLO signals in Prometheus:

```promql
chainwise:frontend_check:request_rate5m
```

```promql
chainwise:frontend_check:error_ratio_rate5m
```

```promql
chainwise:frontend_check:burn_rate5m
```

Check active Chainwise alerts:

```promql
ALERTS{alertname=~"ChainwiseAvailability.*"}
```

Check the frontend request counter grouped by status:

```promql
sum by (service, path, status) (
  rate(chainwise_http_requests_total{
    namespace="chainwise",
    service="frontend",
    path="/check"
  }[5m])
)
```

Check all Chainwise service request counters:

```promql
sum by (service, path, status) (
  rate(chainwise_http_requests_total{namespace="chainwise"}[5m])
)
```

Check Kubernetes pods:

```bash
kubectl -n chainwise get pods -o wide
```

Check recent pod restarts:

```bash
kubectl -n chainwise get pods
```

Check logs for the main request flow:

```bash
kubectl -n chainwise logs deploy/frontend --since=10m
kubectl -n chainwise logs deploy/bike-api --since=10m
kubectl -n chainwise logs deploy/maintenance-api --since=10m
kubectl -n chainwise logs deploy/weather-api --since=10m
kubectl -n chainwise logs deploy/reminder-api --since=10m
kubectl -n chainwise logs deploy/user-api --since=10m
```

Check rollout status:

```bash
kubectl -n chainwise rollout status deploy/frontend
kubectl -n chainwise rollout status deploy/bike-api
kubectl -n chainwise rollout status deploy/maintenance-api
kubectl -n chainwise rollout status deploy/weather-api
kubectl -n chainwise rollout status deploy/reminder-api
kubectl -n chainwise rollout status deploy/user-api
```

## Demo Failure Mode

For the project demo, high error rate can be triggered by enabling controlled failures in `bike-api`:

```bash
kubectl -n chainwise set env deploy/bike-api DEMO_FAIL_RATE=0.1 DEMO_LATENCY_MS=0
kubectl -n chainwise rollout status deploy/bike-api
```

Traffic can be generated against the user-facing endpoint:

```bash
kubectl -n chainwise port-forward svc/frontend 8080:8080
```

```bash
while true; do
  curl -s http://localhost:8080/check > /dev/null
  sleep 1
done
```

## Mitigation

Disable the demo failure mode:

```bash
kubectl -n chainwise set env deploy/bike-api DEMO_FAIL_RATE=0 DEMO_LATENCY_MS=0
kubectl -n chainwise rollout status deploy/bike-api
```

If the issue was caused by a bad rollout, roll back the affected deployment:

```bash
kubectl -n chainwise rollout undo deploy/bike-api
kubectl -n chainwise rollout status deploy/bike-api
```

If another service is failing, roll back the affected deployment:

```bash
kubectl -n chainwise rollout undo deploy/<service-name>
kubectl -n chainwise rollout status deploy/<service-name>
```

If pods are unhealthy, restart the affected deployment:

```bash
kubectl -n chainwise rollout restart deploy/<service-name>
kubectl -n chainwise rollout status deploy/<service-name>
```

## Verification

Verify that `/check` succeeds:

```bash
curl -i http://localhost:8080/check
```

Verify that error ratio returns to zero or near zero:

```promql
chainwise:frontend_check:error_ratio_rate5m
```

Verify that burn rate returns to zero or below alert thresholds:

```promql
chainwise:frontend_check:burn_rate5m
```

Verify that alerts are resolved:

```promql
ALERTS{alertname=~"ChainwiseAvailability.*"}
```

Check Alertmanager UI:

```text
http://localhost:9093
```

Check the SLO dashboard:

```text
https://grafana.dymonyx.ru/d/chainwise-slo/chainwise-slo-overview
```

## Follow-up Actions

After the incident:

- record the incident timeline;
- capture Grafana, Prometheus and Alertmanager evidence;
- document the root cause;
- write a blameless postmortem;
- add action items if the same failure can happen again.