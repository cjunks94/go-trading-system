.PHONY: all build run test bench clean docker-build docker-run lint lint-fix fmt

BINARY_NAME=trading-system
MAIN_PATH=./cmd/server

all: lint test build

build:
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

run:
	go run $(MAIN_PATH)

test:
	go test -v ./...

test-short:
	go test -short -v ./...

bench:
	go test -bench=. -benchmem ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...
	goimports -w .

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

vet:
	go vet ./...

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-build:
	docker build -t go-trading-system .

docker-run:
	docker-compose up --build

docker-down:
	docker-compose down -v

# Install dev tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
