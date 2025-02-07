# Makefile for building, running, and testing the project

BINARY=mock-api

.PHONY: all build run test clean

all: build

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test ./...

clean:
	rm -f $(BINARY)
