#!/usr/bin/env bash

set -euo pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
cd $DIR
KIND_PATH=$GOPATH/src/sigs.k8s.io/kind

# First, clone a `kind` repo fork and build it
echo "Installing kind..."
if [ -d "$KIND_PATH" ];
    then
        echo "$KIND_PATH already exists; Please make sure it is using the latest github.com:nilebox/kind fork."
    else
        echo "Downloading kind (nilebox fork) ..."
        git clone git@github.com:nilebox/kind.git $KIND_PATH
fi

# Switch to a fork branch with support for exposed port range
echo "Building kind..."
cd $KIND_PATH
git checkout expose-port-range
./hack/ci/build-all.sh

echo "======================================================================="

# Switch kubectl context to check if cluster exists
export KUBECONFIG="$(kind get kubeconfig-path)"

# Check if cluster exists and create a new one if it's missing
if ! kubectl cluster-info &> /dev/null;
    then
        # Create a v1.12.2 cluster
        echo "Trying to create a new cluster"
        kind create cluster --image kindest/node:v1.12.2
    else
        echo "kind cluster already exists, skipping..."
fi

# Switch kubectl context (in case the path has changed)
export KUBECONFIG="$(kind get kubeconfig-path)"
kubectl cluster-info

# Switch back to the original directory to deploy things
cd $DIR

echo "======================================================================="

# Create CRDs (they need to be created before any instances)
echo "Creating CRDs"
kubectl apply -f ./crd
# Sleep to make sure CRDs were processed by apiextensions-apiserver
sleep 5

echo "======================================================================="

# Create namespaces (they need to be created before any objects inside)
echo "Creating namespaces"
kubectl apply -f ./namespaces.yaml

echo "-----------------------------------------------------------------------"

# Install Contour
echo "Installing Contour into cluster"
kubectl apply -f ./heptio-contour

echo "-----------------------------------------------------------------------"

# Install Prometheus Operator
echo "Installing Prometheus Operator into cluster"
kubectl apply -f ./prometheus-operator

echo "-----------------------------------------------------------------------"

# Install Grafana
echo "Installing Grafana into cluster"
kubectl apply -f ./grafana

echo "-----------------------------------------------------------------------"

# Install Custom Metrics API Server
echo "Installing Custom Metrics API Server into cluster"
./custom-metrics-api/gencerts.sh
./custom-metrics-api/deploy.sh
./custom-metrics-api/cleanup.sh

echo "-----------------------------------------------------------------------"

# Install kanarini controller
echo "Installing kanarini (CanaryDeployment) controller into cluster"
kubectl apply -f ./kanarini

echo "-----------------------------------------------------------------------"

# Install kanarini example app
echo "Installing kanarini example app into cluster"
kubectl apply -f ./kanarini-demo

echo "-----------------------------------------------------------------------"

# Install manual example app
echo "Installing manual example app into cluster"
kubectl apply -f ./kanarini-manual

echo "-----------------------------------------------------------------------"
# Test that Ingress with load balancing works
echo "Kubernetes cluster is ready. To switch kubectl context, run:"
echo "  export KUBECONFIG=\"$(kind get kubeconfig-path)\""
echo ""
echo "Demo:"
echo "- Grafana dashboards:"
echo "    http://localhost:30988/"
echo "- Testing canary service:"
echo "    curl localhost:30980"
echo "- Testing stable service:"
echo "    curl localhost:30981"
echo "- Testing Contour ingress (load balancer routing traffic between canary and stable services):"
echo "    curl --header \"Host: example.com\" localhost:30900"
echo ""
echo "To test rollback try changing a Docker image to nilebox/kanarini-example:2.0 via"
echo "  kubectl edit canarydeployment -n kanarini-demo emoji"
echo ""
echo "To see progress, you can watch events"
echo "  kubectl get event --namespace kanarini-demo --field-selector involvedObject.kind=CanaryDeployment,involvedObject.name=emoji --sort-by='.lastTimestamp'"

