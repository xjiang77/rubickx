.PHONY: test test-py test-js test-go test-java test-java-one test-scala setup clean \
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

# macOS 上 /usr/bin/java 只是个 stub，PATH 里没有真 JDK 时会报 "Unable to locate
# a Java Runtime"。优先用外部已设的 JAVA_HOME（CI 的 setup-java 会设），
# 否则问 java_home，最后退回 IDE 下载到 ~/Library/Java 的那个。
JAVA_HOME ?= $(shell /usr/libexec/java_home 2>/dev/null || \
	ls -d $(HOME)/Library/Java/JavaVirtualMachines/*/Contents/Home 2>/dev/null | sort -r | head -1)
JAVA  := $(if $(JAVA_HOME),$(JAVA_HOME)/bin/java,java)
JAVAC := $(if $(JAVA_HOME),$(JAVA_HOME)/bin/javac,javac)
JAVA_RELEASE ?= 21
JAVAC_FLAGS := --release $(JAVA_RELEASE)

# 一并导出到 PATH，这样测试里 fork 出来的子进程也能找到同一个 JDK。
ifneq ($(JAVA_HOME),)
export PATH := $(JAVA_HOME)/bin:$(PATH)
endif

# Make 里 '#' 会起注释，只能借一个变量把字面量带进 findstring。
HASH := \#

test-java: $(JUNIT_JAR)
	@echo "== Java (JUnit 5) =="
	@mkdir -p .build/java
	@find problems -path '*/java/*.java' > .build/java/sources.txt
	$(JAVAC) $(JAVAC_FLAGS) -cp "$(JUNIT_JAR)" -d .build/java @.build/java/sources.txt
	$(JAVA) -jar "$(JUNIT_JAR)" execute -cp .build/java --scan-classpath \
		--exclude-tag=practice --details=summary

# 只跑一个测试类或一个测试方法，省掉全量扫描：
#   make test-java-one T=TwoSumTest
#   make test-java-one T=TwoSumTest#basic
# --select-* 不受 @Tag 过滤影响，所以练习类也能直接点名跑。
test-java-one: $(JUNIT_JAR)
	@test -n "$(T)" || { echo "用法: make test-java-one T=TwoSumTest[#method]"; exit 1; }
	@mkdir -p .build/java
	@find problems -path '*/java/*.java' > .build/java/sources.txt
	@$(JAVAC) $(JAVAC_FLAGS) -cp "$(JUNIT_JAR)" -d .build/java @.build/java/sources.txt
	@$(JAVA) -jar "$(JUNIT_JAR)" execute -cp .build/java --details=tree \
		$(if $(findstring $(HASH),$(T)),--select-method='$(T)',--select-class='$(T)')

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
	$(JAVAC) $(JAVAC_FLAGS) -cp "$(JUNIT_JAR)" -d .build/java @.build/java/sources.txt
	$(JAVA) -jar "$(JUNIT_JAR)" execute -cp .build/java --scan-classpath \
		--include-tag=practice --details=tree

clean:
	rm -rf .build .cache
