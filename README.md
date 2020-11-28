# Videochat sample for IT infra course

## How-to

### 0. Prerequisites

1. Install minikube: https://kubernetes.io/ru/docs/tasks/tools/install-minikube/
2. Install skaffold: https://skaffold.dev/docs/install/

### 1. Set up environment

```bash
minikube start

# in a separate terminal
minikube tunnel

# in a separate terminal 
minikube dashboard
```

### 2. Deploy application

```bash
./dev.sh
```

### 3. Spin up chrome for testing

```bash
chromium-browser --use-fake-ui-for-media-stream --use-fake-device-for-media-stream --unsafely-treat-insecure-origin-as-secure="http://<Kuber IP>"
```
Where `<Kuber IP>` is the external IP address found in Kubernetes Dashboard next to nginx service.
