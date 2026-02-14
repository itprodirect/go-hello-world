.PHONY: fmt test build run-cli run-server

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './.git/*')

test:
	go test ./...

build:
	mkdir -p bin
	go build -o bin/hello-cli ./cmd/hello-cli
	go build -o bin/hello-server ./cmd/hello-server

run-cli:
	go run ./cmd/hello-cli --name Nick --repeat 3

run-server:
	go run ./cmd/hello-server
