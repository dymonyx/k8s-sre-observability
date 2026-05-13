# SRE Kubernetes Monitoring Project

This document is an English implementation README for the project work already completed. It describes the infrastructure setup, local tooling, Chainwise deployment workflow, Kubernetes manifests, health checks, resource configuration, and end-to-end validation evidence.

The Russian university report can be maintained separately. This README is intended to be a clean English project description that can later be used for GitHub, portfolio review, and interview demonstrations.

---

## Table of Contents

- [SRE Kubernetes Monitoring Project](#sre-kubernetes-monitoring-project)
  - [Table of Contents](#table-of-contents)
  - [1. Kubernetes Cluster Setup](#1-kubernetes-cluster-setup)
  - [2. Local Tooling Setup](#2-local-tooling-setup)
  - [3. SSH and Kubeconfig Access](#3-ssh-and-kubeconfig-access)
  - [4. Application Development and Local Debugging](#4-application-development-and-local-debugging)
  - [5. Project Management](#5-project-management)
  - [6. Container Image Workflow](#6-container-image-workflow)
  - [7. Kubernetes Manifests for Chainwise](#7-kubernetes-manifests-for-chainwise)
  - [8. Kubernetes Health Checks and Resources](#8-kubernetes-health-checks-and-resources)
  - [9. End-to-End Verification](#9-end-to-end-verification)
  - [10. Evidence](#10-evidence)
  - [11. Installing kube-prometheus-stack](#11-installing-kube-prometheus-stack)
    - [Repository structure](#repository-structure)
    - [Helm repository](#helm-repository)
    - [Namespace creation](#namespace-creation)
    - [Helm values validation](#helm-values-validation)
    - [Installation](#installation)
    - [Component verification](#component-verification)
    - [Accessing Prometheus](#accessing-prometheus)
    - [Accessing Grafana](#accessing-grafana)
    - [Accessing Alertmanager](#accessing-alertmanager)
    - [Evidence](#evidence)
  - [12. Prometheus Scraping for Chainwise](#12-prometheus-scraping-for-chainwise)

---

## 1. Kubernetes Cluster Setup

The project uses a self-managed Kubernetes cluster instead of a local-only Kubernetes environment such as `kind` or `k3d`. This makes the lab closer to a real infrastructure setup, where application workloads run on separate cluster nodes and container images must be pulled from a registry.

The cluster was provisioned on virtual machines. The VM configuration was documented during the setup stage and included multiple nodes with dedicated CPU and RAM resources.

Example VM configuration format:

| Node | Role | vCPU | RAM |
|---|---|---:|---:|
| master | control-plane | `<fill in>` | `<fill in>` |
| worker-1 | worker | `<fill in>` | `<fill in>` |
| worker-2 | worker | `<fill in>` | `<fill in>` |

The cluster installation followed this external guide:

```text
https://habr.com/ru/companies/domclick/articles/682364/
```

Kubespray was used to provision the Kubernetes cluster.

Kubespray repository version:

```text
v2.31.0
```

Before running Kubespray, the virtual machines were prepared with basic system configuration:

```bash
sudo apt update
sudo apt install python3-pip -y

sudo sed -i 's/^#net.ipv4.ip_forward=1/net.ipv4.ip_forward=1/' /etc/sysctl.conf
sudo sysctl -p

sudo swapoff -a
```

The expected sysctl output included:

```text
net.ipv4.ip_forward = 1
```

A Kubespray inventory file was created:

```text
inventory/k8s/hosts.yml
```

On the local machine used for Ansible, the required Python and Ansible dependencies were installed:

```bash
declare -a IPS=(93.77.164.63 93.77.162.161 93.77.167.78)

sudo apt update
sudo apt install -y python3.12-venv python3-pip

cd ~/vscode/k8s-sre-observability/kubespray
python3 -m venv venv
source venv/bin/activate

python -m pip install --upgrade pip
python -m pip install -r requirements.txt
```

The cluster creation playbook was executed with privilege escalation:

```bash
ansible-playbook -i inventory/k8s/hosts.yml cluster.yml -b
```

Kubespray is treated as an external provisioning tool and should not be committed as project application code.

---

## 2. Local Tooling Setup

The local development environment was prepared with the following tools:

- Go
- Docker
- kubectl
- Helm
- golangci-lint
- hadolint

Installation references:

```text
https://go.dev/doc/install
https://docs.docker.com/engine/install/ubuntu
https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/
https://golangci-lint.run/docs/welcome/install/
```

Helm was installed locally using the official Helm package repository:

```bash
sudo apt-get install curl gpg apt-transport-https --yes

curl -fsSL https://packages.buildkite.com/helm-linux/helm-debian/gpgkey \
  | gpg --dearmor \
  | sudo tee /usr/share/keyrings/helm.gpg > /dev/null

echo "deb [signed-by=/usr/share/keyrings/helm.gpg] https://packages.buildkite.com/helm-linux/helm-debian/any/ any main" \
  | sudo tee /etc/apt/sources.list.d/helm-stable-debian.list

sudo apt-get update
sudo apt-get install helm
```

Helm is used as a local CLI tool. During installation or upgrade operations, Helm uses the local kubeconfig to communicate with the Kubernetes API server, similarly to `kubectl`.

---

## 3. SSH and Kubeconfig Access

For convenience, SSH aliases were configured in `~/.ssh/config`.

Example:

```sshconfig
Host master
    HostName <public-ip>
    User agonek
    IdentityFile ~/.ssh/id_ed25519
    ServerAliveInterval 60
```

The Kubernetes kubeconfig was copied from the cluster to the local machine and saved as:

```text
~/.kube/config
```

The local `kubectl` client was then used to manage the remote Kubernetes cluster.

In this setup, the Kubernetes API server is accessed through the public IP address, while TLS verification is performed against an address present in the API server certificate.

Example kubeconfig cluster entry:

```yaml
clusters:
- cluster:
    certificate-authority-data: <redacted>
    server: https://<public-ip>:6443
    tls-server-name: <certificate-san-ip>
  name: kubernetes
```

This avoids disabling TLS verification while still allowing local `kubectl` access to the cluster.

The connection was verified with:

```bash
kubectl config current-context
kubectl get nodes
```

---

## 4. Application Development and Local Debugging

The demo application used in this project is Chainwise, a small microservice-based application for bicycle maintenance recommendations.

During application development and debugging, Docker Compose was used for local execution.

The service call chain is:

```text
frontend
  -> bike-api
    -> maintenance-api
      -> weather-api
        -> reminder-api
          -> user-api
```

Each service exposes basic operational endpoints:

```text
/healthz
/readyz
/metrics
```

The `/check` endpoint on the frontend triggers the full recommendation flow and validates that all service-to-service calls are working.

---

## 5. Project Management

Project work is managed using a GitHub Projects Kanban board.

Completed Kubernetes-related tasks include:

- creating Kubernetes manifests for Chainwise services;
- adding health probes and resource configuration;
- verifying the end-to-end flow inside Kubernetes.

The board is used to track work across application, Kubernetes, observability, SLO, alerting, runbook, incident, and reporting stages.

---

## 6. Container Image Workflow

Docker Compose can build images directly from a local Dockerfile. Kubernetes does not build application images from source code.

Therefore, for reproducible deployment into a real Kubernetes cluster, application images are built locally, pushed to Docker Hub, and referenced from Kubernetes manifests through image references.

Example build and push command for one service:

```bash
docker build --build-arg SERVICE=user-api \
  -t agoneek/chainwise-user-api:dev .

docker push agoneek/chainwise-user-api:dev
```

The same workflow is used for all Chainwise services:

```text
agoneek/chainwise-frontend:dev
agoneek/chainwise-bike-api:dev
agoneek/chainwise-maintenance-api:dev
agoneek/chainwise-weather-api:dev
agoneek/chainwise-reminder-api:dev
agoneek/chainwise-user-api:dev
```

Docker Hub repositories are public for this lab, so Kubernetes does not require an `imagePullSecret`.

---

## 7. Kubernetes Manifests for Chainwise

Kubernetes manifests were created to deploy the Chainwise application into the cluster.

The application namespace is:

```text
chainwise
```

The manifest structure is:

```text
deploy/k8s/
  namespace.yaml
  kustomization.yaml
  frontend.yaml
  bike-api.yaml
  maintenance-api.yaml
  weather-api.yaml
  reminder-api.yaml
  user-api.yaml
```

Each service has:

- a `Deployment`;
- a `Service`;
- service-to-service environment variables;
- a named HTTP container port.

Example service dependency configuration:

```text
frontend        -> BIKE_API_URL=http://bike-api:8081
bike-api        -> MAINTENANCE_API_URL=http://maintenance-api:8082
maintenance-api -> WEATHER_API_URL=http://weather-api:8083
weather-api     -> REMINDER_API_URL=http://reminder-api:8084
reminder-api    -> USER_API_URL=http://user-api:8085
```

Kustomize is used to keep the base manifests independent from the concrete Docker Hub image names.

Example image override in `kustomization.yaml`:

```yaml
images:
  - name: chainwise/frontend
    newName: agoneek/chainwise-frontend
    newTag: dev
```

The application is deployed with:

```bash
kubectl apply -k deploy/k8s
```

The deployment is verified with:

```bash
kubectl get pods -n chainwise
kubectl get svc -n chainwise
```

---

## 8. Kubernetes Health Checks and Resources

Production-like health checks and resource configuration were added to all Chainwise pods.

Each service now has:

- `livenessProbe` using `/healthz`;
- `readinessProbe` using `/readyz`;
- CPU and memory requests;
- CPU and memory limits.

The liveness probe checks whether the container process is alive and should be restarted if it becomes unhealthy.

The readiness probe checks whether the pod is ready to receive traffic through the Kubernetes Service.

Example configuration:

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 2
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /readyz
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 2
  failureThreshold: 3

resources:
  requests:
    cpu: 20m
    memory: 32Mi
  limits:
    cpu: 200m
    memory: 128Mi
```

The rollout was applied with:

```bash
kubectl apply -k deploy/k8s
```

Rollout status was verified with:

```bash
kubectl rollout status deployment/frontend -n chainwise
kubectl rollout status deployment/bike-api -n chainwise
kubectl rollout status deployment/maintenance-api -n chainwise
kubectl rollout status deployment/weather-api -n chainwise
kubectl rollout status deployment/reminder-api -n chainwise
kubectl rollout status deployment/user-api -n chainwise
```

The pods remained `Running` and `Ready` after the rollout.

---

## 9. End-to-End Verification

The frontend service was exposed locally using port-forwarding:

```bash
kubectl -n chainwise port-forward svc/frontend 8080:8080
```

Health and readiness were checked through the forwarded port:

```bash
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/readyz
```

Expected response:

```json
{"service":"frontend","status":"ok"}
```

The full recommendation flow was verified through:

```bash
curl -s http://localhost:8080/check
```

The endpoint returned an HTTP 200 response with a recommendation payload.

The response confirmed that the complete service chain worked inside Kubernetes:

```text
frontend -> bike-api -> maintenance-api -> weather-api -> reminder-api -> user-api
```

The frontend was also opened in a browser through:

```text
http://localhost:8080
```

Internal service reachability was checked from inside the cluster using a temporary curl pod:

```bash
kubectl run curl-test \
  -n chainwise \
  --rm -it \
  --restart=Never \
  --image=curlimages/curl \
  -- sh -c '
    curl -s http://frontend:8080/healthz && echo;
    curl -s http://bike-api:8081/healthz && echo;
    curl -s http://maintenance-api:8082/healthz && echo;
    curl -s http://weather-api:8083/healthz && echo;
    curl -s http://reminder-api:8084/healthz && echo;
    curl -s http://user-api:8085/healthz && echo;
  '
```

All services returned health responses with `status: ok` and their corresponding service names.

---

## 10. Evidence

Validation evidence is stored in:

```text
reports/evidence/
```

The following evidence files were created for the Kubernetes end-to-end verification task:

```text
10-pods.txt
10-services.txt
10-deployments.txt
10-frontend-healthz.json
10-frontend-readyz.json
10-check-response.json
10-frontend-metrics.txt
```

The evidence was collected with commands such as:

```bash
kubectl get pods -n chainwise -o wide > reports/evidence/10-pods.txt
kubectl get svc -n chainwise > reports/evidence/10-services.txt
kubectl get deploy -n chainwise > reports/evidence/10-deployments.txt

curl -s http://localhost:8080/healthz > reports/evidence/10-frontend-healthz.json
curl -s http://localhost:8080/readyz > reports/evidence/10-frontend-readyz.json
curl -s http://localhost:8080/check > reports/evidence/10-check-response.json
curl -s http://localhost:8080/metrics > reports/evidence/10-frontend-metrics.txt
```

---

## 11. Installing kube-prometheus-stack

The next project stage was the installation of the observability stack.

The stack was installed using `kube-prometheus-stack`, a Helm chart that provides a production-like Kubernetes monitoring setup.

The installed stack includes:

- Prometheus;
- Grafana;
- Alertmanager;
- Prometheus Operator;
- kube-state-metrics;
- node-exporter;
- default Kubernetes dashboards and alerting rules.

The monitoring stack was installed into a separate namespace:

```text
observability
```

The application workloads run in the separate namespace:

```text
chainwise
```

This separation keeps application components and observability components isolated and makes the cluster structure easier to manage.

### Repository structure

The Helm values and monitoring-related configuration were placed under the `monitoring/` directory:

```text
monitoring/
  kube-prometheus-stack/
    values.yaml
    README.md
  dashboards/
  prometheus-rules/
```

The `values.yaml` file contains custom Helm values for the `kube-prometheus-stack` installation.

The configuration is intentionally lightweight because the project runs in a self-managed lab Kubernetes cluster. The values file defines:

- Prometheus retention period;
- scrape interval;
- rule evaluation interval;
- CPU and memory requests/limits;
- namespace selectors for future `ServiceMonitor`, `PodMonitor`, and `PrometheusRule` resources.

This prepares the project for the next stages:

- Prometheus scraping configuration for Chainwise;
- Grafana service dashboard;
- SLI/SLO definition;
- recording rules;
- burn-rate alerts;
- Alertmanager routing;
- runbooks and incident simulation.

### Helm repository

The Prometheus Community Helm repository was added locally:

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
```

Available chart versions were checked with:

```bash
helm search repo prometheus-community/kube-prometheus-stack --versions | head
```

The selected chart version was:

```text
85.0.2
```

### Namespace creation

The `observability` namespace was created before installing the monitoring stack:

```bash
kubectl create namespace observability --dry-run=client -o yaml | kubectl apply -f -
```

The namespace was verified with:

```bash
kubectl get ns observability
```

### Helm values validation

Before installing the chart, the Helm templates were rendered locally to validate the configuration:

```bash
helm template kps prometheus-community/kube-prometheus-stack \
  --namespace observability \
  -f monitoring/kube-prometheus-stack/values.yaml > /tmp/kps-rendered.yaml
```

The rendered file was checked with:

```bash
ls -lh /tmp/kps-rendered.yaml
```

### Installation

The stack was installed with Helm:

```bash
helm upgrade --install kps prometheus-community/kube-prometheus-stack \
  --namespace observability \
  --version 85.0.2 \
  -f monitoring/kube-prometheus-stack/values.yaml
```

The Helm release name is:

```text
kps
```

A Helm release is an installed instance of a Helm chart. In this project, `kps` is the installed instance of the `kube-prometheus-stack` chart in the `observability` namespace.

The release was verified with:

```bash
helm list -n observability
```

### Component verification

After installation, the observability pods were checked:

```bash
kubectl get pods -n observability
```

The core components were verified separately:

```bash
kubectl get pods -n observability | grep prometheus
kubectl get pods -n observability | grep grafana
kubectl get pods -n observability | grep alertmanager
```

The Kubernetes services created by the chart were checked with:

```bash
kubectl get svc -n observability
```

The workload resources were also checked:

```bash
kubectl get deploy -n observability
kubectl get sts -n observability
```

All core monitoring components were successfully deployed and became available.

### Accessing Prometheus

Prometheus was accessed locally using port-forward:

```bash
kubectl -n observability port-forward svc/kps-kube-prometheus-stack-prometheus 9090:9090
```

Prometheus UI:

```text
http://localhost:9090
```

Prometheus readiness endpoint can be checked with:

```bash
curl -s http://localhost:9090/-/ready
```

Prometheus is responsible for collecting metrics from Kubernetes components and, in the next stage, from Chainwise services. It stores metrics as time series and provides PromQL for querying and alert rule evaluation.

### Accessing Grafana

Grafana was accessed locally using port-forward:

```bash
kubectl -n observability port-forward svc/kps-grafana 3000:80
```

Grafana UI:

```text
http://localhost:3000
```

The Grafana admin password was retrieved from the Kubernetes Secret created by the Helm chart:

```bash
kubectl -n observability get secret kps-grafana \
  -o jsonpath="{.data.admin-password}" | base64 -d; echo
```

Default username:

```text
admin
```

Grafana will be used to build service-level dashboards and SLO dashboards for Chainwise.

### Accessing Alertmanager

Alertmanager was accessed locally using port-forward:

```bash
kubectl -n observability port-forward svc/kps-kube-prometheus-stack-alertmanager 9093:9093
```

Alertmanager UI:

```text
http://localhost:9093
```

Alertmanager readiness endpoint can be checked with:

```bash
curl -s http://localhost:9093/-/ready
```

Alertmanager will be used later for alert grouping, routing, silences, and inhibition.

### Evidence

Verification outputs for this stage were saved under:

```text
reports/evidence/
```

The following evidence files were collected:

```bash
helm list -n observability > reports/evidence/11-helm-list.txt
kubectl get pods -n observability -o wide > reports/evidence/11-observability-pods.txt
kubectl get deploy -n observability > reports/evidence/11-observability-deployments.txt
kubectl get sts -n observability > reports/evidence/11-observability-statefulsets.txt
kubectl get svc -n observability > reports/evidence/11-observability-services.txt
```

Prometheus and Alertmanager readiness checks can be saved with:

```bash
curl -s http://localhost:9090/-/ready > reports/evidence/11-prometheus-ready.txt
curl -s http://localhost:9093/-/ready > reports/evidence/11-alertmanager-ready.txt
```

Grafana UI verification can be saved as a screenshot.

At the end of this stage, Prometheus, Grafana, and Alertmanager were installed and accessible. The next stage is to configure Prometheus scraping for Chainwise services.

## 12. Prometheus Scraping for Chainwise

Prometheus scraping for Chainwise was configured using a `ServiceMonitor`.

The `ServiceMonitor` selects Chainwise Kubernetes Services by the label:

```yaml
monitoring.chainwise.io/scrape: "true"
```

All Chainwise services expose metrics on:

```text
/metrics
```

The ServiceMonitor configuration is stored in:

```text
monitoring/servicemonitors/chainwise-servicemonitor.yaml
```

The ServiceMonitor was applied with:

```bash
kubectl apply -f monitoring/servicemonitors/chainwise-servicemonitor.yaml
```

It was verified with:

```bash
kubectl get servicemonitor -n chainwise
kubectl describe servicemonitor chainwise-services -n chainwise
```

Prometheus UI was opened with:

```bash
kubectl -n observability port-forward svc/kps-kube-prometheus-stack-prometheus 9090:9090
```

Prometheus was checked at:

```text
http://localhost:9090
```

The following PromQL queries were used for verification:

```promql
up{namespace="chainwise"}
```

Checks that Prometheus sees Chainwise targets. Expected result: six targets with value `1`.

```promql
chainwise_http_requests_total
```

Checks that Chainwise request counters are collected.

```promql
sum by (service) (chainwise_http_requests_total{namespace="chainwise"})
```

Shows total collected HTTP requests grouped by Chainwise service.

```promql
sum by (service, path, status) (chainwise_http_requests_total{namespace="chainwise"})
```

Shows request counters grouped by service, endpoint path, and HTTP status code.

```promql
chainwise_http_request_duration_seconds_sum
```

Checks the request duration count metric.

```promql
sum by (service) (rate(chainwise_http_requests_total{namespace="chainwise"}[5m]))
```

Shows request rate per service.

```promql
sum by (service) (rate(chainwise_http_request_duration_seconds_sum{namespace="chainwise"}[5m]))
/
sum by (service) (rate(chainwise_http_request_duration_seconds_count{namespace="chainwise"}[5m]))
```

Calculates average request duration per service.

At this stage, Prometheus successfully discovered all Chainwise services and collected both request counter and request duration metrics.