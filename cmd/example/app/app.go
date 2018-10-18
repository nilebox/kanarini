package app

import (

	"context"
	"flag"
	"fmt"
	"go.uber.org/zap"
	"github.com/nilebox/kanarini/pkg/util/logz"
	app_util "github.com/nilebox/kanarini/pkg/util/app"
	metric_util "github.com/nilebox/kanarini/pkg/util/metric"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/go-chi/chi"
	"net/http"
	"sync"
	"github.com/nilebox/kanarini/pkg/util/middleware"
)

const (
	defaultReturnCode = 200
	defaultServerAddr = ":8080"
	defaultAuxServerAddr = ":9090"
)

type App struct {
	Logger             *zap.Logger
	PrometheusRegistry metric_util.PrometheusRegistry
	ResponseCode       int

	// Address to listen on
	// Defaults to port 8080
	ServerAddr string

	// Address for auxiliary server to listen on
	// Defaults to port 9090
	AuxServerAddr string

	Debug bool
}

func NewFromFlags(flagset *flag.FlagSet, arguments []string) (*App, error) {
	a := App{
	}

	logEncoding := flagset.String("log-encoding", "json", `Sets the logger's encoding. Valid values are "json" and "console".`)
	loggingLevel := flagset.String("log-level", "info", `Sets the logger's output level.`)

	flagset.StringVar(&a.ServerAddr,"addr", defaultServerAddr, "Port to listen on")
	flagset.StringVar(&a.AuxServerAddr,"aux-addr", defaultAuxServerAddr, "Auxiliary port to listen on")
	flagset.IntVar(&a.ResponseCode,"return-code", defaultReturnCode, "Return code for HTTP requests")
	flagset.BoolVar(&a.Debug,"debug", false, "Enable debug mode")

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
		w.WriteHeader(a.ResponseCode)
		_, err := w.Write([]byte(fmt.Sprintf(`{ "responseCode": "%v" }`, a.ResponseCode)))
		if err != nil {
			a.Logger.Warn("failed to write response body")
		}
	}
	return http.HandlerFunc(fn)
}
