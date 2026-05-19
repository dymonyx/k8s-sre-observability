# Runbook: Chainwise High Latency

## Meaning

This runbook is used when the Chainwise latency SLO for the main recommendation flow is degraded.

Related alerts:

```text
ChainwiseLatencyHighPage
ChainwiseLatencyHighTicket
```

The affected user-facing endpoint is:

```text
frontend /check
```

The latency SLO target is:

```text
The average duration of successful frontend /check requests should stay below 500 ms during normal operation.
```

The current latency SLI is based on average request duration because the current metrics do not expose histogram buckets.

## Impact

Users may still receive maintenance recommendations, but the application becomes slow.

Possible user-visible symptoms:

- the `/check` request takes longer than expected;
- the UI feels slow or unresponsive;
- users may retry requests;
- slow backend dependencies may affect the full recommendation flow.

## Quick Triage

Check current latency:

```promql
chainwise:frontend_check:latency_seconds_avg5m
```

Check latency alerts:

```promql
ALERTS{alertname=~"ChainwiseLatency.*"}
```

Check request rate:

```promql
chainwise:frontend_check:request_rate5m
```

Check backend service request durations:

```promql
sum by (service, path) (
  rate(chainwise_http_request_duration_seconds_sum{namespace="chainwise"}[5m])
)
/
sum by (service, path) (
  rate(chainwise_http_requests_total{namespace="chainwise"}[5m])
)
```

Check pods and restarts:

```bash
kubectl -n chainwise get pods -o wide
```

Check logs for slow services:

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

For the project demo, latency degradation can be triggered by adding artificial latency to `bike-api`:

```bash
kubectl -n chainwise set env deploy/bike-api DEMO_LATENCY_MS=1200 DEMO_FAIL_RATE=0
kubectl -n chainwise rollout status deploy/bike-api
```

Expose the frontend locally:

```bash
kubectl -n chainwise port-forward svc/frontend 8080:8080
```

Generate traffic:

```bash
while true; do
  curl -s http://localhost:8080/check > /dev/null
  sleep 1
done
```

## Mitigation

Disable the demo latency mode:

```bash
kubectl -n chainwise set env deploy/bike-api DEMO_LATENCY_MS=0 DEMO_FAIL_RATE=0
kubectl -n chainwise rollout status deploy/bike-api
```

If the issue was caused by a bad rollout, roll back the affected deployment:

```bash
kubectl -n chainwise rollout undo deploy/bike-api
kubectl -n chainwise rollout status deploy/bike-api
```

If another service is slow, restart or roll back the affected deployment:

```bash
kubectl -n chainwise rollout restart deploy/<service-name>
kubectl -n chainwise rollout status deploy/<service-name>
```

or:

```bash
kubectl -n chainwise rollout undo deploy/<service-name>
kubectl -n chainwise rollout status deploy/<service-name>
```

Check node pressure if latency affects multiple services:

```bash
kubectl top nodes
kubectl top pods -n chainwise
```

## Verification

Verify that `/check` responds successfully:

```bash
curl -i http://localhost:8080/check
```

Verify that latency returns below the SLO target:

```promql
chainwise:frontend_check:latency_seconds_avg5m
```

Expected normal value:

```text
below 0.5 seconds
```

Verify that latency alerts are resolved:

```promql
ALERTS{alertname=~"ChainwiseLatency.*"}
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
- document which service introduced latency;
- write a blameless postmortem;
- consider adding histogram buckets for future percentile-based latency SLOs.