package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/nilebox/kanarini/cmd/kanarini/app"
	app_util "github.com/nilebox/kanarini/pkg/util/app"
)

func main() {
	if err := run(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		fmt.Fprintf(os.Stderr, "%#v\n", err) // nolint: gas, errcheck
		os.Exit(1)
	}
}

func run() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	app_util.CancelOnInterrupt(ctx, cancelFunc)

	return runWithContext(ctx)
}

func runWithContext(ctx context.Context) error {
	a, err := app.NewFromFlags("canary-deployment-controller", flag.CommandLine, os.Args[1:])
	if err != nil {
		return err
	}
	a.Logger.Info("Application starting")
	err = a.Run(ctx)
	a.Logger.Info("Application quit")
	return err
}
