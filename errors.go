package services

import (
	"errors"
	"strings"
)

var (
	// ErrStartCancelledBySignal is returned when Runner.Run receives a shutdown signal while starting the list of
	// Resource and Server.
	ErrStartCancelledBySignal = errors.New("start cancelled by signal")
)

type MultiErrors []error

func (errs MultiErrors) Error() string {
	var r strings.Builder
	for idx, err := range errs {
		if idx > 0 {
			r.WriteString(", ")
		}
		r.WriteString(err.Error())
	}
	return r.String()
}
