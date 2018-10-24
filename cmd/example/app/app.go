package app

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"sync"

	"github.com/go-chi/chi"
	app_util "github.com/nilebox/kanarini/pkg/util/app"
	"github.com/nilebox/kanarini/pkg/util/logz"
	metric_util "github.com/nilebox/kanarini/pkg/util/metric"
	"github.com/nilebox/kanarini/pkg/util/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	defaultErrorRate     = 0.5
	defaultServerAddr    = ":8080"
	defaultAuxServerAddr = ":9090"
)

type App struct {
	Logger             *zap.Logger
	PrometheusRegistry metric_util.PrometheusRegistry
	ErrorRate          float64

	// Address to listen on
	// Defaults to port 8080
	ServerAddr string

	// Address for auxiliary server to listen on
	// Defaults to port 9090
	AuxServerAddr string

	Debug bool
}

func NewFromFlags(flagset *flag.FlagSet, arguments []string) (*App, error) {
	a := App{}

	logEncoding := flagset.String("log-encoding", "json", `Sets the logger's encoding. Valid values are "json" and "console".`)
	loggingLevel := flagset.String("log-level", "info", `Sets the logger's output level.`)

	flagset.StringVar(&a.ServerAddr, "addr", defaultServerAddr, "Port to listen on")
	flagset.StringVar(&a.AuxServerAddr, "aux-addr", defaultAuxServerAddr, "Auxiliary port to listen on")
	flagset.Float64Var(&a.ErrorRate, "error-rate", defaultErrorRate, "Error rate for HTTP requests")
	flagset.BoolVar(&a.Debug, "debug", false, "Enable debug mode")

	err := flagset.Parse(arguments)
	if err != nil {
		return nil, err
	}

	a.Logger = logz.LoggerStr(*loggingLevel, *logEncoding)

	a.PrometheusRegistry = prometheus.NewPedanticRegistry()

	return &a, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.Logger.Sync() // nolint: errcheck
	// unhandled error above, but we are terminating anyway

	router := chi.NewRouter()
	server := &http.Server{
		Addr:           a.ServerAddr,
		MaxHeaderBytes: 1 << 20,
		Handler:        router,
	}
	middleware.Register(a.PrometheusRegistry)
	router.Use(middleware.MonitorRequest)
	router.Handle("/", a.handler())

	// Auxiliary server
	auxServer := app_util.AuxServer{
		Logger:   a.Logger,
		Addr:     a.AuxServerAddr,
		Gatherer: a.PrometheusRegistry,
		IsReady:  func() bool { return true },
		Debug:    a.Debug,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		err := auxServer.Run(ctx)
		if err != nil {
			a.Logger.Sugar().Errorf("auxServer error %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		err := server.ListenAndServe()
		if err != nil {
			a.Logger.Sugar().Errorf("server error %v", err)
		}
	}()

	wg.Wait()
	return nil
}

func (a *App) handler() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := a.generateResponseCode()
		w.WriteHeader(status)
		_, err := w.Write([]byte(fmt.Sprintf(`{ "responseCode": "%v" }`, status)))
		if err != nil {
			a.Logger.Warn("failed to write response body")
		}
	}
	return http.HandlerFunc(fn)
}

func (a *App) generateResponseCode() int {
	num := float64(rand.Intn(101))
	target := a.ErrorRate * 100
	if num > target {
		return http.StatusOK
	} else {
		return http.StatusInternalServerError
	}
}
