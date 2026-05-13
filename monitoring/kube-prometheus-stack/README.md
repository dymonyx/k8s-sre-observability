# kube-prometheus-stack

This directory contains Helm values and operational notes for installing `kube-prometheus-stack`.

The stack is installed into the `observability` namespace and provides:

- Prometheus
- Grafana
- Alertmanager
- Prometheus Operator
- kube-state-metrics
- node-exporter

## Add Helm repository

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm search repo prometheus-community/kube-prometheus-stack --versions | head
```

## Create namespace

```bash
kubectl create namespace observability --dry-run=client -o yaml | kubectl apply -f -
```

## Validate Helm values

```bash
helm template kps prometheus-community/kube-prometheus-stack \
  --namespace observability \
  -f monitoring/kube-prometheus-stack/values.yaml > /tmp/kps-rendered.yaml
```

## Install

```bash
helm upgrade --install kps prometheus-community/kube-prometheus-stack \
  --namespace observability \
  --version 85.0.2 \
  -f monitoring/kube-prometheus-stack/values.yaml
```

## Verify

```bash
helm list -n observability
kubectl get pods -n observability
kubectl get svc -n observability
kubectl get deploy -n observability
kubectl get sts -n observability
```

Check core components:

```bash
kubectl get pods -n observability | grep prometheus
kubectl get pods -n observability | grep grafana
kubectl get pods -n observability | grep alertmanager
```

## Access UIs

### Prometheus

```bash
kubectl -n observability port-forward svc/kps-kube-prometheus-stack-prometheus 9090:9090
```

Open:

```text
http://localhost:9090
```

### Grafana

```bash
kubectl -n observability port-forward svc/kps-grafana 3000:80
```

Open:

```text
http://localhost:3000
```

Get admin password:

```bash
kubectl -n observability get secret kps-grafana \
  -o jsonpath="{.data.admin-password}" | base64 -d; echo
```

Default username:

```text
admin
```

### Alertmanager

```bash
kubectl -n observability port-forward svc/kps-kube-prometheus-stack-alertmanager 9093:9093
```

Open:

```text
http://localhost:9093
```

## Evidence

```bash
mkdir -p reports/evidence

helm list -n observability > reports/evidence/11-helm-list.txt
kubectl get pods -n observability -o wide > reports/evidence/11-observability-pods.txt
kubectl get deploy -n observability > reports/evidence/11-observability-deployments.txt
kubectl get sts -n observability > reports/evidence/11-observability-statefulsets.txt
kubectl get svc -n observability > reports/evidence/11-observability-services.txt
```

Readiness checks:

```bash
curl -s http://localhost:9090/-/ready > reports/evidence/11-prometheus-ready.txt
curl -s http://localhost:9093/-/ready > reports/evidence/11-alertmanager-ready.txt
```

Grafana UI can be saved as a screenshot.

## Notes

The Helm release name is `kps`.

The stack is intentionally configured with lightweight resource requests and limits in `values.yaml`, because this project runs in a self-managed lab Kubernetes cluster.

Chainwise metrics scraping is configured separately with `ServiceMonitor` resources.
