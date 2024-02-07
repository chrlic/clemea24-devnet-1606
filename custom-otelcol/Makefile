SHELL = /bin/bash

.PHONY: all
all: build run

.PHONY: build
build:
	builder --config=builder-config.yaml

.PHONY: run
run: 
	./build/custom-otelcol --config=config.yaml

.PHONY: test
test:
	cd collector/shared/expressions && go test -v .
	cd collector/shared/contextdb && go test -v .
