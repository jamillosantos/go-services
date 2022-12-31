package services

import (
	"context"
	"os"
)

var (
	// DefaultSignals is the list of signals that the Runner will listen if no listener is specified.
	DefaultSignals = []os.Signal{os.Interrupt}
)

// Service is the abstraction of what minimum signature a service must have.
type Service interface {
	// Name will return a human identifiable name for this service. Ex: Postgresql Connection.
	Name() string
}

// Resource is the interface that must be implemented for resourceServices that its start is NOT cancellable.
type Resource interface {
	Service

	// Start will initialize the resource making it ready for use. This method should block until the resource is ready.
	// It should be implemented on a way so that when Stop is called, this should be cancelled. If that is not possible,
	// Stop should wait until Start finish before proceeding.
	//
	// If the service is successfully started, `nil` should be returned. Otherwise, an error must be returned.
	Start(ctx context.Context) error

	// Stop will release this Resource. If it is called while Start still running, Stop should cancel the Start, or then
	// wait for it to be finished before proceeding.
	//
	// For most implementations it will be blocking and should return only when the service finishes stopping. This is
	// important because the Runner relies on it to proceed to the next Resource.
	//
	// If the service is successfully stopped, `nil` should be returned. Otherwise, an error must be returned.
	Stop(ctx context.Context) error
}

// Server is the interface that must be implemented for resourceServices that its start is NOT cancellable.
type Server interface {
	Service

	// Listen will start the server and should never block.
	//
	// If the services is already listining, this should return an error ErrAlreadyListening.
	Listen(ctx context.Context) error

	// Close will stop this service. It should block until all resources are released and the server is properly
	// shutdown.
	//
	// If the services has not started, or is already stopped, this should do nothing and just return nil.
	Close(ctx context.Context) error
}
