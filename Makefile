BINARY := wtt
MODULE := wtt
GO := go

.DEFAULT_GOAL := all
.PHONY: all build test test-cover vet fmt lint clean install

all: build

build:
	$(GO) build -o $(BINARY) ./cmd

test:
	$(GO) test ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

vet:
	$(GO) vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || { gofmt -d .; exit 1; }

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out

install:
	$(GO) install ./cmd
