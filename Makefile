.PHONY: all build run test bench clean docker-build docker-run

BINARY_NAME=trading-system
MAIN_PATH=./cmd/server

all: build

build:
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

run:
	go run $(MAIN_PATH)

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	go fmt ./...
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
