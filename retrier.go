package services

import (
	"context"

	"github.com/cenkalti/backoff/v4"
)

// ResourceServiceRetrier wraps a `Service` in order to provide functionality for retrying in case of its starting process
// fails.
type ResourceServiceRetrier struct {
	service  Resource
	reporter RetrierReporter
	backoff  backoff.BackOff
}

// RetrierBuilder is the helper for building `ResourceServiceRetrier`.
type RetrierBuilder struct {
	backoff  backoff.BackOff
	reporter RetrierReporter
}

// Retrier returns a new `RetrierBuilder` instance.
func Retrier() *RetrierBuilder {
	return &RetrierBuilder{
		backoff: backoff.NewExponentialBackOff(),
	}
}

// Build creates a new `ResourceServiceRetrier` with
func (builder *RetrierBuilder) Build(service Resource) Resource {
	sr := &ResourceServiceRetrier{
		service:  service,
		reporter: builder.reporter,
		backoff:  builder.backoff,
	}
	if configurable, ok := service.(Configurable); ok {
		return struct {
			*ResourceServiceRetrier
			Configurable
		}{
			sr,
			configurable,
		}
	}
	return sr
}

// Backoff set the timeout for the `Retrier`.
func (builder *RetrierBuilder) Backoff(value backoff.BackOff) *RetrierBuilder {
	builder.backoff = value
	return builder
}

// Reporter set the reporter for the `Retrier`.
func (builder *RetrierBuilder) Reporter(value RetrierReporter) *RetrierBuilder {
	builder.reporter = value
	return builder
}

// Name will return a human identifiable name for this service. Ex: Postgresql Connection.
func (retrier *ResourceServiceRetrier) Name() string {
	return retrier.service.Name()
}

// Stop will stop this service.
//
// For most implementations it will be blocking and should return only when the service finishes stopping.
//
// If the service is successfully stopped, `nil` should be returned. Otherwise, an error must be returned.
func (retrier *ResourceServiceRetrier) Stop(ctx context.Context) error {
	return retrier.service.Stop(ctx)
}

// Start implements the logic of starting a service. If it fails, it should use the configuration to retry.
func (retrier *ResourceServiceRetrier) Start(ctx context.Context) error {
	count := 0
	err := backoff.Retry(func() error {
		count++
		if retrier.reporter != nil {
			retrier.reporter.BeforeRetry(ctx, retrier.service, count)
		}
		return retrier.service.Start(ctx)
	}, retrier.backoff)
	return err
}
