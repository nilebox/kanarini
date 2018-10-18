package middleware

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
)

var (
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "request_count",
			Help: "Counter of requests with HTTP code.",
		},
		[]string{"code", "result"},
	)
)

func Register(registerer prometheus.Registerer) {
	registerer.MustRegister(requestCounter)
}

func MonitorRequest(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		delegate := &ResponseWriterDelegator{ResponseWriter: w}
		next.ServeHTTP(delegate, r)
		requestCounter.WithLabelValues(codeToString(delegate), codeToResult(delegate)).Inc()
	}
	return http.HandlerFunc(fn)
}

func codeToString(r *ResponseWriterDelegator) string {
	return fmt.Sprintf("%v", r.status)
}

func codeToResult(r *ResponseWriterDelegator) string {
	statusCode := r.status
	if statusCode >= 200 && statusCode < 300 {
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
