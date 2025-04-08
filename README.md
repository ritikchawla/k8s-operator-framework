# k8s-operator-framework

A custom Kubernetes operator framework for automated management of microservice configurations, featuring GitOps workflow integration and service mesh compatibility.

## Features

- Custom Resource Definition (CRD) for managing microservice configurations
- GitOps workflow integration with ArgoCD
- Service mesh compatibility using Istio
- Automated configuration management
- Kubernetes-native deployment and scaling

## Prerequisites

- Go 1.19+
- Kubernetes cluster 1.22+
- Operator SDK v1.28+
- kubectl
- ArgoCD
- Istio

## Installation

1. Clone the repository:
```bash
git clone https://github.com/ritikchawla/k8s-operator-framework.git
cd k8s-operator-framework
```

2. Install CRDs:
```bash
make install
```

3. Deploy the operator:
```bash
make deploy
```

## Usage

### Basic Microservice Configuration

1. Create a MicroserviceConfig resource:

```yaml
apiVersion: config.ritikchawla.dev/v1alpha1
kind: MicroserviceConfig
metadata:
  name: sample-microservice
spec:
  service:
    image: nginx:latest
    port: 80
    replicas: 3
    resources:
      cpu: "100m"
      memory: "128Mi"
    env:
      - name: ENV
        value: "production"
```

### GitOps Integration

1. Configure ArgoCD for your cluster
2. Add GitOps configuration to your MicroserviceConfig:

```yaml
spec:
  gitops:
    repoURL: "https://github.com/example/microservices-config"
    branch: "main"
    path: "services/sample-microservice"
    autoSync: true
```

### Istio Service Mesh Integration

1. Ensure Istio is installed in your cluster
2. Add Istio configuration to your MicroserviceConfig:

```yaml
spec:
  istio:
    enabled: true
    virtualService:
      hosts:
        - "sample-microservice.example.com"
      gateways:
        - "istio-system/ingress-gateway"
    destinationRule:
      trafficPolicy:
        loadBalancer:
          simple: "ROUND_ROBIN"
```

## Example

A complete example of a MicroserviceConfig resource can be found in `config/samples/config_v1alpha1_microserviceconfig.yaml`.

To apply the example:
```bash
kubectl apply -f config/samples/config_v1alpha1_microserviceconfig.yaml
```

## Development

### Prerequisites

1. Install the required tools:
```bash
brew install go operator-sdk kubectl kubernetes-cli
```

2. Install Operator SDK dependencies:
```bash
make install
```

### Building

1. Build the operator:
```bash
make build
```

2. Run the operator locally:
```bash
make run
```

### Testing

Run tests:
```bash
make test
```

## Architecture

The operator follows the Kubernetes operator pattern and consists of:

1. **Custom Resource Definition (CRD)**: Defines the MicroserviceConfig resource
2. **Controller**: Implements reconciliation logic for:
   - Deployment management
   - Service configuration
   - GitOps integration
   - Istio service mesh setup

### Controller Components

- **Reconciliation Loop**: Ensures desired state matches actual state
- **GitOps Integration**: Manages configuration through Git repositories
- **Service Mesh Configuration**: Handles Istio VirtualService and DestinationRule resources