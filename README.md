[![Go Report Card](https://goreportcard.com/badge/github.com/jamillosantos/go-services)](https://goreportcard.com/report/github.com/jamillosantos/go-services) [![codecov](https://codecov.io/gh/jamillosantos/go-services/branch/master/graph/badge.svg?token=FPOIDZ55TM)](https://codecov.io/gh/jamillosantos/go-services)

**DISCLAIMER: This is a work in progress and is not being used in production yet. Breaking changes might be introduced
without previous warning.**

# go-services

`go-services` is a go library that implement resources and servers and let you start and stop them gracefully. 

## Usage

```go
package main

import (
	"github.com/jamillosantos/go-services"

	"yourproject/internal/resources"
	"yourproject/internal/servers"
)

func main() {
	ctx := context.Background()

	runner := services.NewRunner()
	defer runner.Finish(ctx)

	err := runner.Run(
		ctx,
		resources.Pg,
		resources.Kafka,
		resources.Redis,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("[hit Ctrl+C] to finish ...")
	err := runner.Run(
		ctx,
		servers.Grpc,
		servers.PrometheusMetrics,
	)
	if err != nil {
		panic(err)
	}
}
```

## Ready to use

TODO: Add list of Resource and Server implementations.

## Implementing Resource

**Resources** are dependencies, usually external, needs to be initialized before the main processing of the service.
They are started in sequence and should never block when starting. When the service stops, resources will be stoped
on the reverse order they were started. Examples: Database connections, Amazon SQS, etc. Anything that needs to be
started when the service starts, and stopped when the services is shutting down.

```go
type Resource interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
```

It has a pretty straight forward implementation. The only advise is that `Start` can block until initialized, and after
that it should release the "thread".

If any Resource fails to start, the `ResourceStarter` will stop all previous ones. 

## Implementing Server

**Servers** are dependencies that block the flow of the service. They are initialized in parallel and will block until
the service shuts down. Examples: HTTP servers, gRPC servers, consumers.

```go
type Server interface {
	Name() string
	Listen(ctx context.Context) error
	Close(ctx context.Context) error
}
```