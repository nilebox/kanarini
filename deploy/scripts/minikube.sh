#!/usr/bin/env bash

# set -euo pipefail
# cd "$( dirname "${BASH_SOURCE[0]}" )"
# set -o xtrace

minikube delete
minikube start --kubernetes-version=v1.10.1 --memory=4096 --bootstrapper=kubeadm --extra-config=kubelet.authentication-token-webhook=true --extra-config=kubelet.authorization-mode=Webhook --extra-config=scheduler.address=0.0.0.0 --extra-config=controller-manager.address=0.0.0.0