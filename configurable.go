package services

import (
	"context"
)

// Configurable describes a service that should be loaded before started.
//
// This method will be used direct by `Runner`. Before starting a service, `Runner` will call `Load`
// (if available) before continuing starting the service. If it fails, `Runner.Run` will fail. Otherwise, the
// starting process will continue normally.
type Configurable interface {
	// Load will load the configuration
	Load(ctx context.Context) error
}
