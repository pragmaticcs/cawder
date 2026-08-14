.PHONY: run build tidy

run:
	@go run ./cmd/cawder

build:
	@go build -o bin/cawder ./cmd/cawder

install:
	./install.sh

tidy:
	go mod tidy

test:
	@go test -v ./...
