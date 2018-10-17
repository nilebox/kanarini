package main

import (
	"context"
	"fmt"
	"os"
	"flag"

	app_util "github.com/nilebox/kanarini/pkg/util/app"
	"github.com/nilebox/kanarini/cmd/example/app"
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
	a, err := app.NewFromFlags(flag.CommandLine, os.Args[1:])
	if err != nil {
		return err
	}
	return a.Run(ctx)
}
