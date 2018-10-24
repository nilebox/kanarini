package app

import (
	"time"

	"github.com/atlassian/ctrl"
)

const (
	defaultResyncPeriod = 20 * time.Minute
)

type GenericControllerOptions struct {
	ResyncPeriod time.Duration
	Workers      int
}

func BindGenericControllerFlags(o *GenericControllerOptions, fs ctrl.FlagSet) {
	fs.DurationVar(&o.ResyncPeriod, "resync-period", defaultResyncPeriod, "Resync period for informers")
	fs.IntVar(&o.Workers, "workers", 2, "Number of workers that handle events from informers")
}
