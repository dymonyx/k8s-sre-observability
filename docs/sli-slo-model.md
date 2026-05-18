# Chainwise SLI/SLO Model

This document defines the user-centric SLI/SLO model for the Chainwise recommendation flow.

The goal of this model is to measure the reliability of the most important user-facing path in the application, not only the health of individual Kubernetes pods or services.

---

## Critical User Journey

The critical user journey for Chainwise is:

```text
A user opens the Chainwise frontend, requests a bicycle maintenance check, and receives a maintenance recommendation.
```

This journey is important because it represents the main value of the application: helping a cyclist understand what maintenance actions are currently recommended for their bicycle.

The full request flow is:

```text
browser
  -> frontend /check
    -> bike-api /bike/check
      -> maintenance-api /maintenance/recommendation
        -> weather-api /weather/risk
          -> reminder-api /reminders/next
            -> user-api /users/preferences
```

This request flow was verified using a shared `X-Request-ID`.

Evidence:

```text
reports/evidence/14-check-request-flow-logs.txt
```

The evidence shows that a single request to `frontend /check` propagates through all Chainwise backend services and returns a successful recommendation response.

If any service in this chain fails, the user may not receive a maintenance recommendation. For this reason, the SLO model is based on the frontend endpoint that represents the complete user-facing flow.

---

## Primary Endpoint

The primary endpoint for SLI/SLO measurement is:

```text
frontend /check
```

This endpoint was selected because:

- it is directly triggered by the user;
- it represents the main Chainwise recommendation flow;
- it depends on the backend service chain;
- it exposes failures that affect the user experience;
- it is already covered by Prometheus metrics.

The endpoint is measured using the following metric labels:

```text
namespace="chainwise"
service="frontend"
path="/check"
```

---

## Available Metrics

Chainwise currently exposes the following HTTP metrics:

```text
chainwise_http_requests_total
chainwise_http_request_duration_seconds_sum
```

The request counter includes the following labels:

```text
service
method
path
status
```

These labels allow the project to calculate availability and average latency for the main user-facing endpoint.

Current metrics support:

- request counting;
- request grouping by service, path, method, and status;
- 5xx error ratio calculation;
- average request duration calculation.

Current metrics do not expose histogram buckets. Therefore, this version of the latency SLO uses average request duration instead of p95 or p99 latency.

---

## Availability SLI

The availability SLI measures the percentage of successful requests to the main recommendation flow.

A successful request is defined as a request to `frontend /check` that does not return a `5xx` status code.

Server-side failures are counted as bad events because they indicate that Chainwise could not successfully serve the user request.

### Availability SLI Formula

```text
successful /check requests / total /check requests
```

### PromQL

```promql
sum(rate(chainwise_http_requests_total{
  namespace="chainwise",
  service="frontend",
  path="/check",
  status!~"5.."
}[5m]))
/
sum(rate(chainwise_http_requests_total{
  namespace="chainwise",
  service="frontend",
  path="/check"
}[5m]))
```

### Error Ratio

The error ratio shows the percentage of failed requests and is later used to calculate how quickly the error budget is being consumed.

```promql
sum(rate(chainwise_http_requests_total{
  namespace="chainwise",
  service="frontend",
  path="/check",
  status=~"5.."
}[5m]))
/
sum(rate(chainwise_http_requests_total{
  namespace="chainwise",
  service="frontend",
  path="/check"
}[5m]))
```

---

## Availability SLO

The Chainwise availability SLO is:

```text
99.5% of requests to frontend /check should not return 5xx errors over a rolling 1-day window.
```

This target was selected because Chainwise is a demo SRE project running in a local Kubernetes environment, but the main recommendation flow should still be reliable enough to support meaningful SLO monitoring and burn-rate alerting.

A 1-day SLO window was selected for this project because it is short enough for local validation and demo scenarios, while still representing a real reliability window instead of a very short alerting interval.

### Availability Error Budget

The availability error budget is:

```text
100% - 99.5% = 0.5%
```

This means that during the 1-day SLO window, up to 0.5% of `/check` requests may fail with a `5xx` status before the SLO is considered violated.

Example:

```text
If Chainwise receives 10,000 /check requests in 1 day,
the allowed number of failed 5xx requests is:

10,000 * 0.005 = 50 failed requests
```

If more than 50 requests fail, the availability error budget is exhausted.

---

## Latency SLI

The latency SLI measures how long the main recommendation flow takes from the user's point of view.

Because the current Chainwise metrics expose duration sum but not histogram buckets, latency is measured as average request duration for successful `frontend /check` requests.

### Latency SLI Formula

```text
total duration of successful /check requests / number of successful /check requests
```

### PromQL

```promql
sum(rate(chainwise_http_request_duration_seconds_sum{
  namespace="chainwise",
  service="frontend",
  path="/check",
  status!~"5.."
}[5m]))
/
sum(rate(chainwise_http_requests_total{
  namespace="chainwise",
  service="frontend",
  path="/check",
  status!~"5.."
}[5m]))
```

This query returns the average request duration in seconds.

---

## Latency SLO

The Chainwise latency SLO is:

```text
The average duration of successful frontend /check requests should stay below 500 ms during normal operation.
```

In Prometheus terms:

```text
average /check latency < 0.5 seconds
```

This target is intended for the current project stage and current metrics model. It is strict enough to detect visible degradation, but simple enough to support with the metrics already exposed by the application.

A future improvement would be to expose histogram buckets and define a percentile-based latency SLO, for example:

```text
95% of successful /check requests should complete in less than 500 ms.
```

This improvement is not required for the current task.

---

## SLO Decisions Summary

| Area | Decision |
|---|---|
| Critical user journey | User receives a bicycle maintenance recommendation |
| Primary endpoint | `frontend /check` |
| Availability SLI | Successful `/check` requests divided by total `/check` requests |
| Availability bad events | `/check` requests with `5xx` status |
| Availability SLO | 99.5% successful requests over 1 day |
| Availability error budget | 0.5% failed requests over 1 day |
| Latency SLI | Average duration of successful `/check` requests |
| Latency SLO | Average successful `/check` duration below 500 ms |
| Current limitation | No histogram buckets, so latency uses average duration instead of p95/p99 |

---

## Why This Model Is User-Centric

This SLO model focuses on the user-visible recommendation flow instead of internal service health only.

Pod readiness, CPU usage, memory usage, and individual service health are useful operational signals, but they do not directly answer the most important user-facing question:

```text
Can the user successfully receive a Chainwise maintenance recommendation quickly enough?
```

The selected SLOs answer this question through two symptoms:

1. Availability: does the recommendation request succeed?
2. Latency: does the recommendation request complete fast enough?

These symptoms are suitable for later SLO recording rules, Grafana dashboards, burn-rate alerts, and incident runbooks.