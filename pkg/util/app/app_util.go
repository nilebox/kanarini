package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"net/http"
	"time"
	"sync"

	"github.com/pkg/errors"
)

// CancelOnInterrupt calls f when os.Interrupt or SIGTERM is received.
// It ignores subsequent interrupts on purpose - program should exit correctly after the first signal.
func CancelOnInterrupt(ctx context.Context, f context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-ctx.Done():
		case <-c:
			f()
		}
	}()
}

func StartStopServer(ctx context.Context, srv *http.Server, shutdownTimeout time.Duration) error {
	return StartStopTLSServer(ctx, srv, shutdownTimeout, "", "")
}

func StartStopTLSServer(ctx context.Context, srv *http.Server, shutdownTimeout time.Duration, certFile, keyFile string) error {
	var wg sync.WaitGroup
	defer wg.Wait() // wait for goroutine to shutdown active connections
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if srv.Shutdown(c) != nil {
			srv.Close() // nolint: errcheck,gas,gosec
			// unhandled error above, but we are terminating anyway
		}
	}()

	var err error
	if certFile == "" || keyFile == "" {
		err = srv.ListenAndServe()
	} else {
		err = srv.ListenAndServeTLS(certFile, keyFile)
	}

	if err != http.ErrServerClosed {
		// Failed to start or dirty shutdown
		return errors.WithStack(err)
	}
	// Clean shutdown
	return nil
}
