package services

import (
	"context"
	"os"
	"sync"
)

// Observer will be called Before and After some actions by a `Runner`.
type Observer interface {
	BeforeStart(context.Context, Service)
	AfterStart(context.Context, Service, error)
	BeforeStop(context.Context, Service)
	AfterStop(context.Context, Service, error)
	BeforeLoad(context.Context, Configurable)
	AfterLoad(context.Context, Configurable, error)

	SignalReceived(os.Signal)
}

// runnerObserver is an Observer implementation used by Runner to broadcast the events to the observers.
type runnerObserver struct {
	initObserver sync.Once
	observers    []Observer
}

func (r *runnerObserver) BeforeStart(ctx context.Context, service Service) {
	for _, o := range r.observers {
		o.BeforeStart(ctx, service)
	}
}

func (r *runnerObserver) AfterStart(ctx context.Context, service Service, err error) {
	for _, o := range r.observers {
		o.AfterStart(ctx, service, err)
	}
}

func (r *runnerObserver) BeforeStop(ctx context.Context, service Service) {
	for _, o := range r.observers {
		o.BeforeStop(ctx, service)
	}
}

func (r *runnerObserver) AfterStop(ctx context.Context, service Service, err error) {
	for _, o := range r.observers {
		o.AfterStop(ctx, service, err)
	}
}

func (r *runnerObserver) BeforeLoad(ctx context.Context, configurable Configurable) {
	for _, o := range r.observers {
		o.BeforeLoad(ctx, configurable)
	}
}

func (r *runnerObserver) AfterLoad(ctx context.Context, configurable Configurable, err error) {
	for _, o := range r.observers {
		o.AfterLoad(ctx, configurable, err)
	}
}

func (r *runnerObserver) SignalReceived(signal os.Signal) {
	for _, o := range r.observers {
		o.SignalReceived(signal)
	}
}

func (r *runnerObserver) Add(observer Observer) {
	r.initObserver.Do(func() {
		r.observers = make([]Observer, 0)
	})
	r.observers = append(r.observers, observer)
}
