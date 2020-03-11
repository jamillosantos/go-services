
.PHONY: mod
mod:
	go mod vendor -v


.PHONY: gocyclo
gocyclo:
	go run github.com/fzipp/gocyclo/cmd/gocyclo -over=15 .

.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run

.PHONY: sec
sec:
	go run github.com/securego/gosec/v2/cmd/gosec ./...

.PHONY: test
test: generate lint
	go test ./...

.PHONY: all
all: lint sec test
	@:

.PHONY: generate
generate:
	go generate ./...
