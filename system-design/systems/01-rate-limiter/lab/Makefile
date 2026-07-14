SHELL := /bin/bash

SYSTEM_DESIGN := $(abspath ../../..)
UI := ./ui
BIN := $(CURDIR)/.build/rate-limiter-lab
JAVA_RUNNER := runners/java/RateLimiterRunner.java
JAVA_CLASS := .build/java/RateLimiterRunner.class

.PHONY: setup test test-py test-go test-java test-js test-ui test-race build build-java-runner dev verify verify-redis verify-debug clean

setup:
	python3 -m pip install -q pytest
	cd $(UI) && npm install

test: test-py test-go test-java test-js test-ui

test-py:
	pytest -q runners/python

test-go:
	cd $(SYSTEM_DESIGN) && go test ./systems/01-rate-limiter/lab/server/...

test-java:
	$(MAKE) -C $(SYSTEM_DESIGN) test-java

test-js:
	cd runners/js && node --test

test-ui:
	cd $(UI) && npm test -- --run

test-race:
	cd $(SYSTEM_DESIGN) && go test -race ./systems/01-rate-limiter/lab/server/...

build: build-java-runner
	rm -rf server/internal/webui/dist/assets server/internal/webui/dist/index.html
	cd $(UI) && npm run build
	mkdir -p .build
	cd $(SYSTEM_DESIGN) && go build -o $(BIN) ./systems/01-rate-limiter/lab/server/cmd/lab

build-java-runner: $(JAVA_CLASS)

$(JAVA_CLASS): $(JAVA_RUNNER)
	mkdir -p .build/java
	javac -d .build/java $(JAVA_RUNNER)

dev: build
	LAB_ROOT=$(CURDIR) REDIS_ADDR=$${REDIS_ADDR:-127.0.0.1:6379} $(BIN)

verify: build
	bash scripts/verify-http.sh $(BIN)

verify-redis: build
	bash scripts/verify-redis.sh $(BIN)

verify-debug: build
	bash scripts/verify-debug.sh $(BIN)

clean:
	rm -rf .build ui/node_modules runners/python/__pycache__ server/internal/webui/dist/assets server/internal/webui/dist/index.html
