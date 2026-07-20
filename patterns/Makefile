SHELL := /bin/bash
PYTHON ?= python3

.PHONY: setup lint-catalog test test-pattern test-go-race verify verify-vault clean

JUNIT_VER := 1.10.2
JUNIT_JAR := .cache/junit-platform-console-standalone-$(JUNIT_VER).jar
JACKSON_VER := 2.17.2
JACKSON_ANNOTATIONS := .cache/jackson-annotations-$(JACKSON_VER).jar
JACKSON_CORE := .cache/jackson-core-$(JACKSON_VER).jar
JACKSON_DATABIND := .cache/jackson-databind-$(JACKSON_VER).jar
JAVA_TEST_DEPS := $(JUNIT_JAR) $(JACKSON_ANNOTATIONS) $(JACKSON_CORE) $(JACKSON_DATABIND)

setup: $(JAVA_TEST_DEPS)
	$(PYTHON) -m pip install -q pytest

lint-catalog:
	$(PYTHON) scripts/verify_catalog.py --catalog-only

test-pattern: $(JAVA_TEST_DEPS) lint-catalog
	@test -n "$(PATTERN)" || (echo "PATTERN is required, for example PATTERN=gof.structural.adapter" >&2; exit 2)
	$(PYTHON) scripts/verify_catalog.py --pattern "$(PATTERN)"
	$(PYTHON) scripts/run_pattern.py --pattern "$(PATTERN)"

test: $(JAVA_TEST_DEPS) lint-catalog
	$(PYTHON) scripts/run_pattern.py --all

test-go-race:
	go test -race ./...

verify: $(JAVA_TEST_DEPS) lint-catalog
	$(PYTHON) scripts/verify_catalog.py
	$(PYTHON) scripts/run_pattern.py --all
	go test -race ./...

verify-vault: verify
	@test -n "$(VAULT_ROOT)" || (echo "VAULT_ROOT is required" >&2; exit 2)
	$(PYTHON) scripts/verify_catalog.py --vault-root "$(VAULT_ROOT)"

$(JUNIT_JAR):
	@mkdir -p .cache
	curl -fsSL -o "$@" https://repo1.maven.org/maven2/org/junit/platform/junit-platform-console-standalone/$(JUNIT_VER)/junit-platform-console-standalone-$(JUNIT_VER).jar

$(JACKSON_ANNOTATIONS):
	@mkdir -p .cache
	curl -fsSL -o "$@" https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-annotations/$(JACKSON_VER)/jackson-annotations-$(JACKSON_VER).jar

$(JACKSON_CORE):
	@mkdir -p .cache
	curl -fsSL -o "$@" https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-core/$(JACKSON_VER)/jackson-core-$(JACKSON_VER).jar

$(JACKSON_DATABIND):
	@mkdir -p .cache
	curl -fsSL -o "$@" https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-databind/$(JACKSON_VER)/jackson-databind-$(JACKSON_VER).jar

clean:
	rm -rf .build .cache .pytest_cache
	find . -type d -name __pycache__ -prune -exec rm -rf {} +
