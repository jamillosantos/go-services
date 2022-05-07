package services

import (
	"context"
	"os"
	"sync"

	signals "github.com/jamillosantos/go-os-signals"
)

type Runner struct {
	resourceServices []Resource

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
		resourceServices: make([]Resource, 0),
	}
	for _, opt := range opts {
		opt(manager)
	}

	return manager
}

func stopServers(ctx context.Context, observer Observer, servers []Server) {
	for _, server := range servers {
		err := server.Close(ctx)
		observer.AfterStop(ctx, server, err)
	}
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
	var listener signals.Listener
	if r.listenerBuilder == nil {
		listener = signals.NewListener(DefaultSignals...)
	} else {
		listener = r.listenerBuilder()
	}
	defer listener.Stop()

	ctxSignal, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go func() {
		// Intercepts a signal cancelling the procedure of starting services.
		<-listener.Receive()
		cancelFunc()
	}()

	servers := make([]Server, 0, len(services))
	hasServer := false
	var serversMutex sync.Mutex

	var wgServers sync.WaitGroup

	// Make sure that all servers will be finished
	defer wgServers.Wait()

	// Finish all servers
	defer func() {
		serversMutex.Lock()
		defer serversMutex.Unlock()
		stopServers(ctx, &r.observer, servers)
	}()

	errs := make(chan errPair, len(services))

	// Go through all resourceServices starting one by one.
	serverCount := 0
	for _, service := range services {
		// Check if the starting process was cancelled.
		select {
		case <-ctx.Done():
			if hasServer {
				break
			}
			return ctx.Err()
		case <-ctxSignal.Done():
			if hasServer {
				break
			}
			return ErrStartCancelledBySignal
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
			if hasServer {
				break
			}
			return ctx.Err()
		case <-ctxSignal.Done():
			if hasServer {
				break
			}
			return ErrStartCancelledBySignal
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
			r.resourceServices = append(r.resourceServices, s)
		case Server:
			hasServer = true
			wgServers.Add(1)

			serversMutex.Lock()
			servers = append(servers, s)
			serversMutex.Unlock()

			go func(s Server, idx int) {
				defer wgServers.Done()

				err := s.Listen(ctx)
				if err != nil && err != context.Canceled {
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
		if !hasServer {
			return ctx.Err()
		}
	case <-ctxSignal.Done():
		if !hasServer {
			return ErrStartCancelledBySignal
		}
	default:
		// Not cancelled ...
	}

	if !hasServer {
		return nil
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
	case <-ctxSignal.Done(): // Wait a signal to come in.
		return nil
	case <-ctx.Done(): // the deferred methods will handle this...
		return ctx.Err()
	}
}

// Finish will go through all started resourceServices, in the opposite order they were started, stopping one by one. If any,
// failure is detected, the function will stop leaving some started resourceServices.
func (r *Runner) Finish(ctx context.Context) (errResult error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for i := len(r.resourceServices) - 1; i >= 0; i-- {
		service := r.resourceServices[i]
		r.observer.BeforeStop(ctx, service)
		err := service.Stop(ctx)
		r.observer.AfterStop(ctx, service, err)
		if err != nil {
			return err
		}
		r.resourceServices = r.resourceServices[:len(r.resourceServices)-1]
	}
	return nil
}

type errPair struct {
	idx int
	err error
}
