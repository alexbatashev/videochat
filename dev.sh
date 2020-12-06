#!/bin/sh

# This script exists, because https://github.com/GoogleContainerTools/skaffold/issues/4641

kubectl --context minikube create -f common
skaffold build --tag=latest
skaffold dev --tag="latest"
