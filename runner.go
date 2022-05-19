package services

import (
	"context"
	"os"
	"sync"
	"time"

	signals "github.com/jamillosantos/go-os-signals"
)

type Runner struct {
	servicesM sync.Mutex
	services  []Service

	wgServices sync.WaitGroup

	observer        runnerObserver
	listenerBuilder func() signals.Listener
}

type StarterOption = func(*Runner)

// WithReporter is a StarterOption that will set the signal listener instance of a Runner.
func WithReporter(reporter Observer) StarterOption {
	return func(manager *Runner) {
		manager.observer.Add(reporter)
	}
}

// WithObserver is a StarterOption that will add an observer instance on the list.
func WithObserver(observer Observer) StarterOption {
	return func(manager *Runner) {
		manager.observer.Add(observer)
	}
}

// WithListenerBuilder is a StarterOption that will set the signal listener instance of a Runner.
func WithListenerBuilder(builder func() signals.Listener) StarterOption {
	return func(manager *Runner) {
		manager.listenerBuilder = builder
	}
}

// WithSignals is a StarterOption that will setup a listener builder that create a listener with the given signals.
func WithSignals(ss ...os.Signal) StarterOption {
	return func(manager *Runner) {
		manager.listenerBuilder = func() signals.Listener {
			return signals.NewListener(ss...)
		}
	}
}

// NewRunner creates a new instance of Runner.
//
// If a listener is not defined, it will create one based on DefaultSignals.
func NewRunner(opts ...StarterOption) *Runner {
	manager := &Runner{
		services: make([]Service, 0),
	}
	for _, opt := range opts {
		opt(manager)
	}

	return manager
}

// Run goes through all given Service instances trying to start them. This function only supports Resource or Server
// instances (subset of Service). Then, it goes through all of them starting each one.
//
// Resource instances are initialized by calling Resource.Start, respecting the given order, only one at a time. If only
// Resource instances are passed, this function will not block and Run can be called many times (not thread-safe).
//
// Server instances are initialized by invoking a new goroutine that calls the Server.Listen. So, the order is not be
// guaranteed and all Server starts at once. Then, Run blocks until all server are closed and it can happen in two
// cases: when a specified os.Signal is received (check WithListenerBuilder or WithSignals for more information) or when
// the given ctx is cancelled. Either cases the Run will gracefully stop all Server instances that were initialized
// (by calling Server.Close).
//
// Important: Resource instances will not be stopped when the a os.Signal is received or the ctx is cancelled. For that,
// you should call Runner.Finish.
//
// If you need to cancel the Run method. You can use the context.WithCancel applied to the given ctx.
//
// Whenever this function exists, all given Server instances will be closed by using Server.Close. Then, it will wait
// until the Server.Listen finished.
func (r *Runner) Run(ctx context.Context, services ...Service) (errResult error) {
	errs := make(chan errPair, len(services))

	// Go through all resourceServices starting one by one.
	serverCount := 0
	for _, service := range services {
		// Check if the starting process was cancelled.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Not cancelled ...
		}

		// If the service is configurable
		if srv, ok := service.(Configurable); ok {
			r.observer.BeforeLoad(ctx, srv)
			errResult = srv.Load(ctx)
			r.observer.AfterLoad(ctx, srv, errResult)
			if errResult != nil {
				return
			}
		}

		// Loading configuration can take a long time. Then, check if the starting process was cancelled again.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Not cancelled ...
		}

		r.observer.BeforeStart(ctx, service)

		switch s := service.(type) {
		case Resource:
			errResult = s.Start(ctx)
			r.observer.AfterStart(ctx, service, errResult)
			if errResult != nil {
				return
			}
			r.servicesM.Lock()
			r.wgServices.Add(1)
			r.services = append(r.services, s)
			r.servicesM.Unlock()
		case Server:
			r.wgServices.Add(1)

			r.addService(s)

			go func(s Server, idx int) {
				defer r.wgServices.Done()

				err := s.Listen(ctx)
				if err != nil && err != context.Canceled {
					r.removeService(s)
					errs <- errPair{
						idx,
						err,
					}
				}
			}(s, serverCount)
			serverCount++
		}
	}

	// Loading configuration can take a long time. Then, check if the starting process was cancelled again.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Not cancelled ...
	}

	errMulti := make(MultiErrors, serverCount)

	select {
	case ep := <-errs:
		if ep.err != nil {
			errMulti[ep.idx] = ep.err
		}
		close(errs)
		for ep := range errs {
			errMulti[ep.idx] = ep.err
		}
		return errMulti
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Second):
		return nil
	}
}

// Wait will block until all servers are stopped.
func (r *Runner) Wait(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		r.wgServices.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-ctx.Done():
		return
	}
}

// Finish will go through all started resourceServices, in the opposite order they were started, stopping one by one. If any,
// failure is detected, the function will stop leaving some started resourceServices.
func (r *Runner) Finish(ctx context.Context) (errResult error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	var err error
	r.servicesM.Lock()
	for i := len(r.services) - 1; i >= 0; i-- {
		service := r.services[i]
		r.observer.BeforeStop(ctx, service)
		switch s := service.(type) {
		case Resource:
			err = s.Stop(ctx)
			r.wgServices.Done()
		case Server:
			err = s.Close(ctx)
		}
		r.observer.AfterStop(ctx, service, err)
		if err != nil {
			return err
		}
		r.services = r.services[:len(r.services)-1]
	}
	r.servicesM.Unlock()

	return nil
}

func (r *Runner) addService(s Service) {
	r.servicesM.Lock()
	r.services = append(r.services, s)
	r.servicesM.Unlock()
}

func (r *Runner) removeService(s Service) {
	r.servicesM.Lock()
	for i, server := range r.services {
		if server == s {
			r.services = append(r.services[:i], r.services[i+1:]...)
			break
		}
	}
	r.servicesM.Unlock()
}

type errPair struct {
	idx int
	err error
}
