# Kanarini, Canary Deployment Controller for Kubernetes [![Build Status](https://travis-ci.com/nilebox/kanarini.svg?branch=master)](https://travis-ci.com/nilebox/kanarini)

## Overview

Kanarini is a CRD controller that implements canary deployment strategy by
creating two Deployment objects - "canary" and "stable", and validating the
healthiness of Deployments via calls to Custom Metrics API (https://github.com/kubernetes/metrics).

Kanarini introduces a new Kubernetes resource, `CanaryDeployment`, that reflects
the structure of standard Deployment with extra configuration for canary/stable
deployment tracks.

## Prerequisites

### Custom Metrics API

Kanarini is using [Custom Metrics API](https://github.com/kubernetes/metrics#custom-metrics-api) to fetch metrics, and one of its implementations
is required to be installed in the cluster.
For example, if you are using Prometheus for metrics collection, you can install
[Prometheus Adapter](https://github.com/DirectXMan12/k8s-prometheus-adapter).

## Getting started

The easiest way to see Kanarini in action is to follow the [Quickstart](#quickstart) guide.

### Add Kanarini to your cluster

Run:
```bash
kubectl apply -f ./deploy/crd-canarydeployment.yaml
kubectl apply -f ./deploy/namespace.yaml
kubectl apply -f ./deploy/serviceaccount.yaml
kubectl apply -f ./deploy/rbac.yaml
kubectl apply -f ./deploy/deployment.yaml
```

This command:
- Registers a `CanaryDeployment` CustomResourceDefinition
- Creates a new namespace `kanarini` with one instance of Kanarini in the namespace
- Grants Kanarini permissions to manage Deployment objects in the cluster

### Usage example

To create an instance of `CanaryDeployment`, run:
```bash
kubectl apply -f ./deploy/example.yaml
```

This command will create a `CanaryDeployment` in the `kanarini-example` namespace:
```bash
$ kubectl get canarydeployments -n kanarini-example emoji
NAME    AGE
emoji   22s
```
```yaml
apiVersion: kanarini.nilebox.github.com/v1alpha1
kind: CanaryDeployment
metadata:
  name: emoji
  namespace: kanarini-demo
spec:
  selector:
    matchLabels:
      app: emoji
  template:
    metadata:
      labels:
        app: emoji
    spec:
      containers:
      - name: emoji
        image: nilebox/kanarini-example:1.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
  tracks:
    canary:
      replicas: 1
      labels:
        track: canary
      metricsCheckDelaySeconds: 60
      metrics:
      - type: Object
        object:
          describedObject:
            kind: "Service"
            name: "emoji-canary"
          metric:
            name: "failure_rate_1m"
          target:
            type: Value
            value: 0.1
    stable:
      replicas: 3
      labels:
        track: stable
```

As a result, you will have two `Deployment` objects created and managed by Kanarini.

## Quickstart

To see Kanarini in action, run a "Quickstart" script that bootstraps
a local Kubernetes cluster (https://github.com/kubernetes-sigs/kind), installs
additional components (Kanarini controller, 
[Heptio Contour](https://github.com/heptio/contour), [Prometheus Operator](https://github.com/coreos/prometheus-operator),
[Prometheus Adapter](https://github.com/DirectXMan12/k8s-prometheus-adapter), Grafana)
and deploys a demo application with `CanaryDeployment` resource, services and ingress.

First, run
```bash
./demo/quickstart.sh
```

Once the script has successfully finished its execution, setup `kubectl` context:
```bash
export KUBECONFIG="$(kind get kubeconfig-path)"
```

To see Grafana dashboards, open in your browser http://localhost:30988/ 
(default username/password is `admin`/`admin`).

To see a working example, you can use `curl` or browser with address http://localhost:30900.
Note that you need to set the header `Host: example.com` in order for Contour
to forward requests to underlying "canary" and "stable" services. To achieve that,
you can use a Firefox browser extension [Header Editor](https://addons.mozilla.org/en-US/firefox/addon/header-editor/).

Alternatively, you could open service URLs directly:
- canary: http://localhost:30980
- stable: http://localhost:30981

Note that in that case you won't get weighted load balancing, as you will be sending 
requests directly to services without ingress.

To test happy path scenario, change Docker image in 
[canarydeployments.yaml](https://github.com/nilebox/kanarini/blob/master/demo/kanarini-demo/canarydeployments.yaml)
to `nilebox/kanarini-example:3.0`. The change will be first applied to a "canary"
`Deployment`, and after a successful metric check it will be propagated to "stable" `Deployment`.

To test non-happy path, change Docker image in 
[canarydeployments.yaml](https://github.com/nilebox/kanarini/blob/master/demo/kanarini-demo/canarydeployments.yaml)
to `nilebox/kanarini-example:2.0`. The change will be initially applied to a "canary"
`Deployment`, but after failing metric check it will be rolled back.
