.PHONY: test test-py test-go test-go-race test-java test-js test-ui build-lab setup clean

# 一键跑全部四种语言；UI 是独立构建面，CI 通过 test-ui/build-lab 验证。
test: test-py test-go test-java test-js

# 安装可在仓库内直接运行的依赖（Python）
setup:
	pip install -q pytest
	cd systems/01-rate-limiter/lab/ui && npm install

test-py:
	@echo "== Python (pytest) =="
	pytest -q

test-go:
	@echo "== Go (go test) =="
	go test ./...

test-go-race:
	@echo "== Go (go test -race) =="
	go test -race ./...

JUNIT_VER := 1.10.2
JUNIT_JAR := .cache/junit-platform-console-standalone-$(JUNIT_VER).jar

test-java: $(JUNIT_JAR)
	@echo "== Java (JUnit 5) =="
	@mkdir -p .build/java
	@find systems -path '*/java/*.java' > .build/java/sources.txt
	javac -cp "$(JUNIT_JAR)" -d .build/java @.build/java/sources.txt
	java -jar "$(JUNIT_JAR)" execute -cp .build/java --scan-classpath --details=summary

test-js:
	@echo "== JavaScript (node:test) =="
	cd systems/01-rate-limiter/lab/runners/js && node --test

test-ui:
	@echo "== Rate Limiter Lab UI (Vitest) =="
	cd systems/01-rate-limiter/lab/ui && npm test -- --run

build-lab:
	$(MAKE) -C systems/01-rate-limiter/lab build

$(JUNIT_JAR):
	@mkdir -p .cache
	curl -fSL -o "$(JUNIT_JAR)" https://repo1.maven.org/maven2/org/junit/platform/junit-platform-console-standalone/$(JUNIT_VER)/junit-platform-console-standalone-$(JUNIT_VER).jar

clean:
	rm -rf .build .cache systems/01-rate-limiter/lab/.build
