package app

import (
	"context"
	"io"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	defaultMaxRequestDuration = 15 * time.Second
	shutdownTimeout           = defaultMaxRequestDuration
	readTimeout               = 1 * time.Second
	writeTimeout              = defaultMaxRequestDuration
	idleTimeout               = 1 * time.Minute
)

type AuxServer struct {
	Logger *zap.Logger
	// Name is the name of the application.
	Name     string
	Addr     string
	Gatherer prometheus.Gatherer
	IsReady  func() bool
	Debug    bool
}

func (a *AuxServer) Run(ctx context.Context) error {
	if a.Addr == "" {
		<-ctx.Done()
		return nil
	}
	srv := &http.Server{
		Addr:         a.Addr,
		Handler:      a.constructHandler(),
		WriteTimeout: writeTimeout,
		ReadTimeout:  readTimeout,
		IdleTimeout:  idleTimeout,
	}
	return StartStopServer(ctx, srv, shutdownTimeout)
}

func (a *AuxServer) constructHandler() *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Timeout(defaultMaxRequestDuration), a.setServerHeader)
	router.NotFound(pageNotFound)

	router.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(a.Gatherer, promhttp.HandlerOpts{}))
	router.Get("/healthz/ping", func(_ http.ResponseWriter, _ *http.Request) {})
	router.Get("/healthz/ready", func(w http.ResponseWriter, _ *http.Request) {
		if !a.IsReady() {
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, "Not ready") // nolint: errcheck, gosec
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	if a.Debug {
		// Enable debug endpoints
		router.HandleFunc("/debug/pprof/", pprof.Index)
		router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	return router
}

func (a *AuxServer) setServerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", a.Name)
		next.ServeHTTP(w, r)
	})
}

func pageNotFound(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}
