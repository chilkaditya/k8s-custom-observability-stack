# Custom Kubernetes Observability Operator

A Kubernetes Operator built with Go, Kubebuilder, and controller-runtime to automate observability configuration for Kubernetes workloads.

The operator uses a custom resource, `CustomObservabilityStack`, to manage observability resources such as Grafana dashboard ConfigMaps, Prometheus alerting rules, and ServiceMonitors.

## Features

- **Custom Resource Definition (CRD):** Configure observability through Kubernetes-native YAML manifests.
- **Grafana dashboards:** Automatically create or reconcile dashboard ConfigMaps.
- **Prometheus alerting:** Manage PrometheusRule resources through the operator.
- **Service monitoring:** Manage ServiceMonitor resources for Prometheus scraping.
- **Reconciliation:** Continuously reconcile managed resources toward the desired state.
- **Helm deployment:** Install and manage the operator using a published Helm chart.
- **Kubernetes RBAC:** Define permissions required by the controller, including leader election.

## Architecture

```text
       User / Application Team
                  |
                  | Applies Custom Resource
                  v
       CustomObservabilityStack
                  |
                  v
       Custom Observability Operator
             (Go + Kubebuilder)
                  |
          Reconciliation Loop
                  |
          +-------+--------+
          |       |        |
          v       v        v
      ConfigMap Prometheus ServiceMonitor
      Dashboard    Rule
          |       |        |
          v       v        v
       Grafana  Prometheus Prometheus
```

## Technology Stack

| Component | Technology |
|---|---|
| Language | Go |
| Operator Framework | Kubebuilder |
| Kubernetes APIs | controller-runtime |
| Dashboards | Grafana |
| Metrics and alerting | Prometheus |
| Service discovery | Prometheus Operator |
| Packaging | Helm |
| Container image | Docker |
| Development environment | Kubernetes / Minikube |

## Prerequisites

Ensure the following tools and components are available:

- Kubernetes cluster
- `kubectl`
- Helm 3 or a compatible Helm version supporting OCI registries
- A running Prometheus Operator installation if you want to use `PrometheusRule` and `ServiceMonitor`
- Grafana and Prometheus if you want to visualize dashboards, collect metrics, and evaluate alerts

Verify your Kubernetes connection:

```bash
kubectl cluster-info
kubectl get nodes
helm version
```

Check whether the Prometheus Operator CRDs are installed:

```bash
kubectl get crd prometheusrules.monitoring.coreos.com
kubectl get crd servicemonitors.monitoring.coreos.com
```

If these CRDs are missing, install a compatible Prometheus Operator distribution before enabling the corresponding features.

## Installation

### Option 1: Install directly from the published Helm OCI chart

The chart is published to Docker Hub as an OCI artifact.

```bash
helm install observability \
  oci://registry-1.docker.io/apu10/custom-observability-operator \
  --version 0.1.0 \
  --namespace observability-system \
  --create-namespace
```

This command installs the chart version `0.1.0` into the `observability-system` namespace.

> **Note:** The operator image must also be accessible to your Kubernetes cluster. For a private image repository, configure image pull credentials.

### Option 2: Install from the source repository

Clone the project:

```bash
git clone <YOUR_GITHUB_REPOSITORY_URL>
cd k8s-observability-op
```

Install the chart from the local directory:

```bash
helm install observability \
  ./charts/custom-observability-operator \
  --namespace observability-system \
  --create-namespace
```

Replace `<YOUR_GITHUB_REPOSITORY_URL>` with the actual Git repository URL.

## Verify the Installation

Check the Helm release:

```bash
helm list -n observability-system
```

Check the deployment:

```bash
kubectl get deployments -n observability-system
```

Check the operator Pod:

```bash
kubectl get pods -n observability-system
```

Check the operator logs:

```bash
kubectl logs \
  -n observability-system \
  deployment/observability-custom-observability-operator-controller-manager \
  --since=5m
```

Confirm that the custom resource definition is installed:

```bash
kubectl get crd customobservabilitystacks.apps.chilkaditya.me
```

## Configure an Observability Stack

After installing the operator, create a `CustomObservabilityStack` custom resource.

Create a file named `observability-stack.yaml`:

```yaml
apiVersion: apps.chilkaditya.me/v1
kind: CustomObservabilityStack
metadata:
  name: nginx-observability
  namespace: default
spec:
  targetDeployment: nginx-deployment
  targetNamespace: default
  cpuThresold: 80
  memoryThresold: 80
  enableDashboard: true
```

**Before applying this example:**

- Ensure that `nginx-deployment` exists in the `default` namespace.
- Confirm that the field names match the installed CRD schema.
- Ensure the required monitoring components and CRDs are installed.

Apply the resource:

```bash
kubectl apply -f observability-stack.yaml
```

Inspect the custom resource:

```bash
kubectl get customobservabilitystacks -A
kubectl describe customobservabilitystack nginx-observability -n default
```

Inspect the managed resources:

```bash
kubectl get configmaps -n default
kubectl get prometheusrules -A
kubectl get servicemonitors -A
```

The exact resources created depend on the operator implementation and the fields configured in the custom resource.

## Configure the Helm Chart

Inspect the chart's available values:

```bash
helm show values \
  oci://registry-1.docker.io/apu10/custom-observability-operator \
  --version 0.1.0
```

To customize installation, create a `values.yaml` file containing overrides supported by the chart.

For example, if the chart exposes these image values:

```yaml
manager:
  image:
    repository: docker.io/apu10/custom-observability-op
    tag: v0.1.0
    pullPolicy: IfNotPresent
```

Use the override only if these keys match the chart's actual `values.yaml` structure.

Install or upgrade with your overrides:

```bash
helm upgrade --install observability \
  oci://registry-1.docker.io/apu10/custom-observability-operator \
  --version 0.1.0 \
  --namespace observability-system \
  --create-namespace \
  -f values.yaml
```

## Upgrade the Operator

After publishing a new compatible chart version, upgrade the release by specifying the new version:

```bash
helm upgrade observability \
  oci://registry-1.docker.io/apu10/custom-observability-operator \
  --version <NEW_CHART_VERSION> \
  --namespace observability-system
```

Review the chart's release notes and values before upgrading production installations.

## Uninstall

Remove the Helm release:

```bash
helm uninstall observability -n observability-system
```

**Important:** uninstalling the release does not necessarily remove custom resources, their managed resources, or CRDs. Review resource ownership and the chart's CRD handling before deleting anything. Avoid deleting shared monitoring resources that other applications depend on.

## Development

To build the operator image locally:

```bash
make docker-build \
  IMG=docker.io/apu10/custom-observability-op:v0.1.0
```

To push the image:

```bash
docker login
make docker-push \
  IMG=docker.io/apu10/custom-observability-op:v0.1.0
```

To validate the Helm chart:

```bash
helm lint charts/custom-observability-operator
helm template observability \
  charts/custom-observability-operator \
  --namespace observability-system
```

To package the chart:

```bash
helm package charts/custom-observability-operator
```

To publish a packaged chart to Docker Hub:

```bash
helm registry login registry-1.docker.io -u apu10

helm push custom-observability-operator-<CHART_VERSION>.tgz \
  oci://registry-1.docker.io/apu10
```

Update `version` in `Chart.yaml` before publishing a new chart release.

## Troubleshooting

### Operator Pod is not starting

```bash
kubectl describe pod -n observability-system <POD_NAME>
kubectl logs -n observability-system <POD_NAME>
```

Check image availability, image pull credentials, resource limits, and scheduling events.

### Leader-election permission errors

The controller's ServiceAccount must have permission to access the required Lease resources.

```bash
kubectl auth can-i get leases \
  --as=system:serviceaccount:observability-system:observability-custom-observability-operator \
  -n observability-system
```

Inspect the installed ClusterRole and ClusterRoleBinding if the command returns `no`.

### PrometheusRule or ServiceMonitor is not created

```bash
kubectl get crd prometheusrules.monitoring.coreos.com
kubectl get crd servicemonitors.monitoring.coreos.com
kubectl logs -n observability-system \
  deployment/observability-custom-observability-operator-controller-manager
```

Verify the CRD availability, RBAC permissions, custom resource configuration, and reconciliation errors.

### Helm installation or upgrade fails

```bash
helm status observability -n observability-system
helm get values observability -n observability-system
helm get manifest observability -n observability-system
```

Review the output and the operator logs before making changes to cluster resources.

## Contributing

Contributions, bug reports, and feature requests are welcome.

1. Fork the repository.
2. Create a feature branch.
3. Make your changes and test them.
4. Open a pull request describing the change.

Please include reproduction steps for bugs and tests for new functionality where appropriate.

## License

Add the project's license and license link here once the repository's licensing choice has been finalized.
