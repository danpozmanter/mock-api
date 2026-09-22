# Build, run, and verify mock-api with the Gossamer toolchain (gos).

BINARY := target/release/mock-api
CONFIG ?= config.yaml
PORT ?= 8080

.PHONY: all build run test check fmt lint smoke clean

all: build

build:
	gos build --release

run:
	gos run src/main.gos --config $(CONFIG) --port $(PORT)

test:
	gos test

check:
	gos check src/main.gos

fmt:
	gos fmt --check

lint:
	gos lint

smoke: build
	scripts/smoke.sh $(BINARY)

clean:
	rm -rf target dist .gos-cache
