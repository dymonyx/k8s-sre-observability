
## Quick Triage

List Chainwise pods:

```bash
kubectl -n chainwise get pods -o wide
```

Describe the affected pod:

```bash
kubectl -n chainwise describe pod <pod-name>
```

Check recent events:

```bash
kubectl -n chainwise get events --sort-by=.lastTimestamp
```

Check container logs:

```bash
kubectl -n chainwise logs <pod-name>
```

If the pod restarted, check previous logs:

```bash
kubectl -n chainwise logs <pod-name> --previous
```

Check readiness and liveness probe configuration:

```bash
kubectl -n chainwise get deploy <deployment-name> -o yaml
```

Check service endpoints:

```bash
kubectl -n chainwise get endpoints
```

Check rollout status:

```bash
kubectl -n chainwise rollout status deploy/<deployment-name>
```

Check resource usage:

```bash
kubectl top pods -n chainwise
kubectl top nodes
```

## Common Causes

Common causes for a pod not being ready:

- readiness probe failure;
- liveness probe failure and repeated restarts;
- application startup problem;
- bad environment variable or config;
- image pull issue;
- resource pressure;
- dependency failure;
- demo failure mode affecting health endpoints.

## Demo Failure Mode

A pod readiness issue can be simulated by applying an invalid or too aggressive demo setting.

Example:

```bash
kubectl -n chainwise set env deploy/bike-api DEMO_FAIL_RATE=0.9
kubectl -n chainwise rollout status deploy/bike-api
```

This may cause the pod to restart or stay not ready if demo failures also affect health or readiness checks.

Use this carefully. For normal SLO alert verification, prefer lower failure rates or controlled latency.

## Mitigation

Disable demo failure modes:

```bash
kubectl -n chainwise set env deploy/bike-api DEMO_FAIL_RATE=0 DEMO_LATENCY_MS=0
kubectl -n chainwise rollout status deploy/bike-api
```

Restart the affected deployment:

```bash
kubectl -n chainwise rollout restart deploy/<deployment-name>
kubectl -n chainwise rollout status deploy/<deployment-name>
```

Roll back if the problem was introduced by a bad rollout:

```bash
kubectl -n chainwise rollout undo deploy/<deployment-name>
kubectl -n chainwise rollout status deploy/<deployment-name>
```

If the pod is stuck, inspect and delete it so the ReplicaSet creates a new one:

```bash
kubectl -n chainwise delete pod <pod-name>
```

If resources are exhausted, check requests, limits and node pressure:

```bash
kubectl describe node <node-name>
kubectl top nodes
kubectl top pods -n chainwise
```

## Verification

Verify all Chainwise pods are ready:

```bash
kubectl -n chainwise get pods
```

Expected result:

```text
READY 1/1 for all Chainwise pods
```

Verify endpoints exist:

```bash
kubectl -n chainwise get endpoints
```

Verify the main user-facing endpoint works:

```bash
kubectl -n chainwise port-forward svc/frontend 8080:8080
```

```bash
curl -i http://localhost:8080/check
```

Verify SLO signals recover:

```promql
chainwise:frontend_check:error_ratio_rate5m
```

```promql
chainwise:frontend_check:latency_seconds_avg5m
```

Verify no related alerts are firing:

```promql
ALERTS{namespace="chainwise"}
```

## Follow-up Actions

After the incident:

- record which pod was not ready;
- capture pod events and logs;
- document whether the issue was caused by config, rollout, resources or probes;
- add action items to prevent recurrence;
- update probes or demo failure controls if health endpoints were affected unintentionally.