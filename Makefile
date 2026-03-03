# rubickx — progressive AI agent tutorial
# Usage: make help

-include .env
export

S ?=

.DEFAULT_GOAL := help

.PHONY: help run run-trpc test test-unit test-api check check-trpc web-dev web-install setup

help:  ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

run:  ## Run a session REPL (go): make run S=06
	@test -n "$(S)" || { echo "Usage: make run S=01"; exit 1; }
	cd go && go run ./s$(S)*/

run-trpc:  ## Run a session REPL (trpc-agent-go): make run-trpc S=01
	@test -n "$(S)" || { echo "Usage: make run-trpc S=01"; exit 1; }
	cd trpc-agent-go && go run ./s$(S)*/

test: check check-trpc test-unit test-api  ## Run all tests (check → unit → api)

test-unit:  ## Run offline Python unit tests
	python3 tests/test_unit.py

test-api:  ## Run API connectivity test (needs ANTHROPIC_API_KEY)
	python3 tests/test_s01_verify.py

check:  ## Compile-check and vet all go sessions
	@cd go && for s in s*/; do echo "build $$s"; go build -o /dev/null ./"$$s" || exit 1; done
	@cd go && go vet ./...

check-trpc:  ## Compile-check and vet all trpc-agent-go sessions
	@cd trpc-agent-go && for s in s*/; do echo "build $$s"; go build -o /dev/null ./"$$s" || exit 1; done
	@cd trpc-agent-go && go vet ./...

web-dev:  ## Start the Next.js dev server
	cd deps/learn-claude-code/web && npm run dev

web-install:  ## Install web app dependencies
	cd deps/learn-claude-code/web && npm install

setup:  ## Initial project setup (submodule + deps)
	git submodule update --init --recursive
	cd go && go mod download
	cd trpc-agent-go && go mod download
	$(MAKE) web-install
