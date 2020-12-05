#!/bin/sh

# This script exists, because https://github.com/GoogleContainerTools/skaffold/issues/4641

kubectl --context minikube create -f common
helm repo add bitnami https://charts.bitnami.com/bitnami
skaffold build --tag=latest --kube-context minikube
skaffold run --tag="latest" --kube-context minikube
