#!/usr/bin/env bash

set -euo pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
cd $DIR

kubectl apply -f apiservice.yaml
kubectl apply -f cm-adapter-serving-certs.yaml
kubectl apply -f deployment.yaml
kubectl apply -f rbac.yaml
kubectl apply -f service.yaml
kubectl apply -f serviceaccount.yaml
