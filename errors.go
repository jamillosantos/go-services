package services

import "errors"

var (
	// ErrStartCancelledBySignal is returned when Runner.Run receives a shutdown signal while starting the list of
	// Resource and Server.
	ErrStartCancelledBySignal = errors.New("start cancelled by signal")
)
