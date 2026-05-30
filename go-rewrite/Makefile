.PHONY: build run clean test

build:
	go build -o bin/meridian ./cmd/meridian

run: build
	./bin/meridian start

test:
	go test ./...

clean:
	rm -rf bin/

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...
