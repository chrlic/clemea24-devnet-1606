SHELL = /bin/bash

.PHONY: build
build:
	builder --config=builder-config.yaml

.PHONY: run
run: build
	./build/custom-otelcol --config=config.yaml

.PHONY: test
test:
	cd collector/shared/expressions && go test -v .
	cd collector/shared/contextdb && go test -v .
