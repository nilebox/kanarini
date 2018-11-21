# Kanarini, Canary Deployment Controller for Kubernetes

## Overview

Kanarini is a CRD controller that implements canary deployment strategy by
creating two Deployment objects - "canary" and "stable", and validating the
healthiness of Deployments via calls to Custom Metrics API (https://github.com/kubernetes/metrics).

Kanarini introduces a new Kubernetes resource, `CanaryDeployment`, that reflects
the structure of standard Deployment with extra configuration for canary/stable
deployment tracks.

## Getting started

The easiest way to try Kanarini is to run a "Quickstart" script that bootstraps
a local Kubernetes cluster (https://github.com/kubernetes-sigs/kind), installs
additional components (Kanarini controller, 
[Heptio Contour](https://github.com/heptio/contour), [Prometheus Operator](https://github.com/coreos/prometheus-operator),
[Prometheus Adapter](https://github.com/DirectXMan12/k8s-prometheus-adapter), Grafana)
and deploys a demo application with `CanaryDeployment` resource, services and ingresses.

First, run
```bash
./deploy/quickstart.sh
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
[canarydeployments.yaml](https://github.com/nilebox/kanarini/blob/master/deploy/kanarini-demo/canarydeployments.yaml)
to `nilebox/kanarini-example:3.0`. The change will be first applied to a "canary"
Deployment, and then to "stable" Deployment.

To test non-happy path, change Docker image in 
[canarydeployments.yaml](https://github.com/nilebox/kanarini/blob/master/deploy/kanarini-demo/canarydeployments.yaml)
to `nilebox/kanarini-example:2.0`. The change will be applied to a "canary"
Deployment, but after failing metric check it will be rolled back.
