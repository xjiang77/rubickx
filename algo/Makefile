.PHONY: test test-py test-js test-go test-java test-scala setup clean \
        practice practice-go practice-java

# 一键跑全部 5 种语言的测试
test: test-py test-js test-go test-java test-scala

# 安装可在仓库内直接运行的依赖（Python + JS）
setup:
	pip install -q pytest
	npm install --silent

test-py:
	@echo "== Python (pytest) =="
	pytest -q

test-js:
	@echo "== JavaScript (jest) =="
	npx jest

test-go:
	@echo "== Go (go test) =="
	go test ./...

JUNIT_VER := 1.10.2
JUNIT_JAR := .cache/junit-platform-console-standalone-$(JUNIT_VER).jar

test-java: $(JUNIT_JAR)
	@echo "== Java (JUnit 5) =="
	@mkdir -p .build/java
	@find problems -path '*/java/*.java' > .build/java/sources.txt
	javac -cp "$(JUNIT_JAR)" -d .build/java @.build/java/sources.txt
	java -jar "$(JUNIT_JAR)" execute -cp .build/java --scan-classpath \
		--exclude-tag=practice --details=summary

$(JUNIT_JAR):
	@mkdir -p .cache
	curl -sSL -o "$(JUNIT_JAR)" https://repo1.maven.org/maven2/org/junit/platform/junit-platform-console-standalone/$(JUNIT_VER)/junit-platform-console-standalone-$(JUNIT_VER).jar

test-scala:
	@echo "== Scala (scala-cli + munit) =="
	@for d in problems/*/scala; do echo "-- $$d"; scala-cli test "$$d" || exit 1; done

# ---------------------------------------------------------------------------
# 练习模式：跑 *Practice 的空白练习（自己手写实现的那一份）
# 默认的 `make test` 不会碰它们，所以练习没写完也不会把主测试搞红。
#   - Go 靠 build tag：练习文件带 //go:build practice
#   - Java 靠 JUnit tag：练习测试类带 @Tag("practice")
# ---------------------------------------------------------------------------
practice: practice-go practice-java

practice-go:
	@echo "== Go 练习 (build tag: practice) =="
	go test -tags practice ./...

practice-java: $(JUNIT_JAR)
	@echo "== Java 练习 (@Tag: practice) =="
	@mkdir -p .build/java
	@find problems -path '*/java/*.java' > .build/java/sources.txt
	javac -cp "$(JUNIT_JAR)" -d .build/java @.build/java/sources.txt
	java -jar "$(JUNIT_JAR)" execute -cp .build/java --scan-classpath \
		--include-tag=practice --details=tree

clean:
	rm -rf .build .cache
