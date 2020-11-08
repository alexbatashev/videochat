# Videochat sample for IT infra course

How-to:
```bash
docker-compose up --build
chromium-browser --use-fake-ui-for-media-stream --use-fake-device-for-media-stream
# Or
chrome.exe --use-fake-ui-for-media-stream --use-fake-device-for-media-stream
```

k8s how-to:
```bash
minikube start --cpus=6
minikube dashboard
# in a separate terminal
minikube tunnel
# in a separate terminal
kubectl apply -f k8s
```
