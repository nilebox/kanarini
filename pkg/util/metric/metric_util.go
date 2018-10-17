package metric


import (
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusRegistry interface {
	prometheus.Registerer
	prometheus.Gatherer
}
