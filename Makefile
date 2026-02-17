.PHONY: fmt vet test test-cover bench build clean run-cli run-server run-healthcheck

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './.git/*')

vet:
	go vet ./...

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "View in browser: go tool cover -html=coverage.out"

bench:
	go test -bench=. -benchmem ./...

build:
	mkdir -p bin
	go build -o bin/hello-cli ./cmd/hello-cli
	go build -o bin/hello-server ./cmd/hello-server
	go build -o bin/healthcheck ./cmd/healthcheck

clean:
	rm -rf bin coverage.out

run-cli:
	go run ./cmd/hello-cli --name Nick --repeat 3 --style formal

run-server:
	go run ./cmd/hello-server --config config.example.json

run-healthcheck:
	go run ./cmd/healthcheck --targets targets.example.json --workers 4
