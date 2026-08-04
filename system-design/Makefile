.PHONY: test test-py test-go test-go-race test-java test-java-one test-js test-ui build-lab setup clean

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

# macOS 上 /usr/bin/java 只是个 stub，PATH 里没有真 JDK 时会报 "Unable to locate
# a Java Runtime"。优先用外部已设的 JAVA_HOME（CI 的 setup-java 会设），
# 否则问 java_home，最后退回 IDE 下载到 ~/Library/Java 的那个。
JAVA_HOME ?= $(shell /usr/libexec/java_home 2>/dev/null || \
	ls -d $(HOME)/Library/Java/JavaVirtualMachines/*/Contents/Home 2>/dev/null | sort -r | head -1)
JAVA  := $(if $(JAVA_HOME),$(JAVA_HOME)/bin/java,java)
JAVAC := $(if $(JAVA_HOME),$(JAVA_HOME)/bin/javac,javac)
JAVA_RELEASE ?= 21
JAVAC_FLAGS := --release $(JAVA_RELEASE)

# RateLimiterRunnerTest 用 ProcessBuilder("java", ...) fork 子 JVM，靠 PATH 找
# java，所以只设 $(JAVA) 不够，得把 JDK 导出到 PATH 让子进程继承。
ifneq ($(JAVA_HOME),)
export PATH := $(JAVA_HOME)/bin:$(PATH)
endif

# Make 里 '#' 会起注释，只能借一个变量把字面量带进 findstring。
HASH := \#

test-java: $(JUNIT_JAR)
	@echo "== Java (JUnit 5) =="
	@mkdir -p .build/java
	@find systems -path '*/java/*.java' > .build/java/sources.txt
	$(JAVAC) $(JAVAC_FLAGS) -cp "$(JUNIT_JAR)" -d .build/java @.build/java/sources.txt
	$(JAVA) -jar "$(JUNIT_JAR)" execute -cp .build/java --scan-classpath --details=summary

# 只跑一个测试类或一个测试方法：
#   make test-java-one T=TokenBucketTest
#   make test-java-one T=TokenBucketTest#cap
test-java-one: $(JUNIT_JAR)
	@test -n "$(T)" || { echo "用法: make test-java-one T=TokenBucketTest[#method]"; exit 1; }
	@mkdir -p .build/java
	@find systems -path '*/java/*.java' > .build/java/sources.txt
	@$(JAVAC) $(JAVAC_FLAGS) -cp "$(JUNIT_JAR)" -d .build/java @.build/java/sources.txt
	@$(JAVA) -jar "$(JUNIT_JAR)" execute -cp .build/java --details=tree \
		$(if $(findstring $(HASH),$(T)),--select-method='$(T)',--select-class='$(T)')

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
