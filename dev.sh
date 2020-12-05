#!/bin/sh

# This script exists, because https://github.com/GoogleContainerTools/skaffold/issues/4641

kubectl --context minikube create -f common
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add kafka-topics-ui https://dhiatn.github.io/helm-kafka-topics-ui
skaffold build --tag=latest
skaffold dev --tag="latest"
