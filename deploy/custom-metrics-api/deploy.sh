#!/usr/bin/env bash

set -euo pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
cd $DIR

if [ -f cm-adapter-serving-certs.yaml ];
    then
        kubectl apply -f cm-adapter-serving-certs.yaml
    else
        echo "Secret YAML doesn't exist, skipping"
fi

kubectl apply -f apiservice.yaml
kubectl apply -f configmaps.yaml
kubectl apply -f deployment.yaml
kubectl apply -f rbac.yaml
kubectl apply -f service.yaml
kubectl apply -f serviceaccount.yaml
