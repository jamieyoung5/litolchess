.PHONY: all build run test clean fmt vet check

all: check

build:
	go build -o bin/littolchess ./cmd

run:
	go run ./cmd

test:
	go test ./...

clean:
	rm -rf bin

fmt:
	go fmt ./...

vet:
	go vet ./...

check: fmt vet test
