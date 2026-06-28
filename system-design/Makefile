.PHONY: test test-py test-go test-java setup clean

# 一键跑全部三种语言
test: test-py test-go test-java

# 安装可在仓库内直接运行的依赖（Python）
setup:
	pip install -q pytest

test-py:
	@echo "== Python (pytest) =="
	pytest -q

test-go:
	@echo "== Go (go test) =="
	go test ./...

JUNIT_VER := 1.10.2
JUNIT_JAR := .cache/junit-platform-console-standalone-$(JUNIT_VER).jar

test-java: $(JUNIT_JAR)
	@echo "== Java (JUnit 5) =="
	@mkdir -p .build/java
	@find systems -path '*/java/*.java' > .build/java/sources.txt
	javac -cp "$(JUNIT_JAR)" -d .build/java @.build/java/sources.txt
	java -jar "$(JUNIT_JAR)" execute -cp .build/java --scan-classpath --details=summary

$(JUNIT_JAR):
	@mkdir -p .cache
	curl -sSL -o "$(JUNIT_JAR)" https://repo1.maven.org/maven2/org/junit/platform/junit-platform-console-standalone/$(JUNIT_VER)/junit-platform-console-standalone-$(JUNIT_VER).jar

clean:
	rm -rf .build .cache
