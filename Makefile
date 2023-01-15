.DEFAULT_GOAL := build-all

LDFLAGS=-ldflags="-X 'github.com/weeveiot/weeve-agent/internal/model.Version=$(shell date +%Y.%m.%d) ($(shell git rev-parse --short HEAD))'"

build:
	go build $(LDFLAGS) -o bin/weeve-agent ./cmd/agent/agent.go
.PHONY: build

build-x86:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/weeve-agent-linux-amd64 ./cmd/agent/agent.go
.PHONY: build-x86

build-arm:
	GOOS=linux GOARCH=arm GOARM=7 go build $(LDFLAGS) -o bin/weeve-agent-linux-arm-v7 ./cmd/agent/agent.go
.PHONY: build-arm

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/weeve-agent-darwin ./cmd/agent/agent.go
.PHONY: build-darwin

cross:
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o bin/weeve-agent-linux-amd64    ./cmd/agent/agent.go
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o bin/weeve-agent-linux-arm64    ./cmd/agent/agent.go
	GOOS=linux   GOARCH=arm   go build $(LDFLAGS) -o bin/weeve-agent-linux-arm      ./cmd/agent/agent.go
.PHONY: cross

build-all: build-arm build-x86 build-darwin
.PHONY: build-all
