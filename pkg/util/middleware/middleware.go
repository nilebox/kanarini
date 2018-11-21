package middleware

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_count",
			Help: "Counter of requests with result.",
		},
		[]string{"version", "result"},
	)
	totalRequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_count_total",
			Help: "Counter of total requests",
		},
		[]string{"version"},
	)
)

func Register(registerer prometheus.Registerer, version string) *MonitorMiddleware {
	// Initialize metrics to zero to prevent missing metrics
	// See https://www.robustperception.io/existential-issues-with-metrics
	requestCounter.WithLabelValues(version, flagToResult(true))
	requestCounter.WithLabelValues(version, flagToResult(false))
	totalRequestCounter.WithLabelValues(version)

	registerer.MustRegister(requestCounter)
	registerer.MustRegister(totalRequestCounter)

	return &MonitorMiddleware{
		Version: version,
	}
}

type MonitorMiddleware struct {
	Version string
}

func (m *MonitorMiddleware) MonitorRequest(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.URL != nil && r.URL.String() == "/favicon.ico" {
			// Don't instrument favicon requests
			next.ServeHTTP(w, r)
			return
		}
		delegate := &ResponseWriterDelegator{ResponseWriter: w}
		next.ServeHTTP(delegate, r)
		requestCounter.WithLabelValues(m.Version, codeToResult(delegate)).Inc()
		totalRequestCounter.WithLabelValues(m.Version).Inc()
	}
	return http.HandlerFunc(fn)
}

func codeToResult(r *ResponseWriterDelegator) string {
	statusCode := r.status
	if statusCode >= 200 && statusCode < 300 {
		return flagToResult(true)
	}
	return flagToResult(false)
}

func flagToResult(success bool) string {
	if success {
		return "success"
	}
	return "failure"
}

// ResponseWriterDelegator interface wraps http.ResponseWriter to additionally record content-length, status-code, etc.
type ResponseWriterDelegator struct {
	http.ResponseWriter

	status      int
	written     int64
	wroteHeader bool
}

func (r *ResponseWriterDelegator) WriteHeader(code int) {
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseWriterDelegator) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)
	return n, err
}
