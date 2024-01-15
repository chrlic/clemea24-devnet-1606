SHELL = /bin/bash

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 ./ocb --config=builder-config.yaml

.PHONY: lab-linux-collector
lab-linux-collector:
	scp -o ProxyCommand='nc -x 127.0.0.1:1080 %h %p' otelcol-dev/otelcol-dev 192.168.33.31:otel-col/

.PHONY: lab-linux-config
lab-linux-config:
	set -o allexport && . ./secrets-lab.txt && cat coll-config.yaml | envsubst > config.yaml
	sed -i "" "/socks5/d" config.yaml
	scp -o ProxyCommand='nc -x 127.0.0.1:1080 %h %p' config.yaml 192.168.33.31:otel-col/
	scp -o ProxyCommand='nc -x 127.0.0.1:1080 %h %p' -r conf 192.168.33.31:otel-col/

.PHONY: lab-linux
lab-linux: build-linux lab-linux-collector  lab-linux-config

.PHONY: build
build:
	./ocb --config=builder-config.yaml

.PHONY: run
run: build
	set -o allexport && . ./secrets.txt && cat coll-config.yaml | envsubst > config.yaml
	./otelcol-dev/otelcol-dev --config=config.yaml

.PHONY: rundev
rundev: build
	set -o allexport && . ./secrets.txt && cat coll-config-dev-net.yaml | envsubst > config.yaml
	./otelcol-dev/otelcol-dev --config=config.yaml

.PHONY: test
test:
	cd collector/shared/expressions && go test -v .
	cd collector/shared/contextdb && go test -v .
