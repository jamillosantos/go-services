package services

import (
	"context"
	"os"
)

// Reporter will be called Before and After some actions by a `Runner`.
type Reporter interface {
	BeforeStart(context.Context, Service)
	AfterStart(context.Context, Service, error)
	BeforeStop(context.Context, Service)
	AfterStop(context.Context, Service, error)
	BeforeLoad(context.Context, Configurable)
	AfterLoad(context.Context, Configurable, error)

	SignalReceived(os.Signal)
}

type RetrierReporter interface {
	Reporter
	BeforeRetry(context.Context, Service, int)
}
