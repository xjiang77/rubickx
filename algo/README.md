# algo — 五语言算法刷题板块

用 **Python / Go / JavaScript / Java / Scala** 五种语言实现同一道题,横向对比各语言的实现方式、惯用数据结构与现代最佳实践。配套 FAANG 4 周冲刺计划(见仓库 outputs 里的落地执行计划)。

> 设计理念与 rubickx 一致:**同一问题多语言重写,在差异里吃透每种语言的设计取向。**

## 目录结构(按题并排)

```
algo/
├── Makefile                 # make test / test-py / test-js / test-go / test-java / test-scala
├── go.mod                   # 单一 Go module: rubickx/algo
├── package.json             # JS: jest
├── pytest.ini               # Python: pytest
├── PROGRESS.md              # 进度追踪表
└── problems/
    └── 0001-two-sum/
        ├── NOTES.md         # ⭐ 五语言对比 + 最佳实践(中文)
        ├── python/  two_sum.py        + two_sum_test.py
        ├── go/      twosum.go         + twosum_test.go        (package twosum)
        ├── js/      twoSum.js         + twoSum.test.js
        ├── java/    TwoSum.java       + TwoSumTest.java       (默认包,类名唯一)
        └── scala/   TwoSum.scala      + TwoSumTest.scala      (scala-cli + munit)
```

每题文件夹里 5 种实现并排,`NOTES.md` 是核心——讲清这道题在 5 种语言里**怎么写、为什么这么写、最近实践是什么**。

## 怎么跑测试

```bash
cd algo
make setup        # 安装 Python(pytest)+ JS(jest)依赖
make test         # 跑全部 5 种语言
# 或单独跑：
make test-py      # pytest
make test-js      # jest
make test-go      # go test ./...
make test-java    # javac + JUnit 5 console
make test-scala   # scala-cli test 每个 problems/*/scala
```

### 工具链版本(最近实践)

| 语言 | 版本 | 测试框架 | 备注 |
|---|---|---|---|
| Python | 3.10+ | pytest | 用 `list[int]` 内建泛型注解、`collections` |
| Go | 1.21+ | 标准库 `testing` | 单 module,表驱动测试 |
| JavaScript | Node 18+ | jest 29 | CommonJS(`require`/`module.exports`) |
| Java | 17+(LTS,代码兼容 11) | JUnit 5.10 | 默认包 + console launcher |
| Scala | 3.4 | munit 1.0 | 用 `scala-cli`,非 sbt |

> 已在沙箱验证:**Python + JS 全部测试通过**。Go / Java / Scala 需本机对应工具链(`go` / JDK 含 `javac` / [scala-cli](https://scala-cli.virtuslab.org/))。

## 几个工程决策(为什么这么搭)

- **按题并排而非按语言分目录**:核心诉求是"对比同一题的 5 种实现",并排最直观;代价是各语言工具链要适配非标准布局(见下)。
- **Java 用默认包 + 唯一类名 + JUnit console launcher**:Maven/Gradle 强约定 `src/main/java` 与"包=目录",和"按题并排"冲突。改用 `javac` 直接编译所有 `problems/**/java/*.java` 到一个 classpath,再用 `junit-platform-console-standalone` 扫描运行。代价:类名必须全局唯一(故为 `TwoSum`、`ContainsDuplicate`…)。
- **Scala 用 scala-cli 而非 sbt**:`scala-cli test <dir>` 能直接跑单目录里的脚本式 Scala + 内联依赖指令(`//> using test.dep ...`),是当下写 kata/单文件 Scala 的最佳实践,免去 sbt 的重型工程结构。
- **Python 用 pytest + `--import-mode=importlib`**:每个测试文件把自己所在目录加进 `sys.path` 再 import 同级解法,避免不同题目同名模块互相覆盖。
- **Go 单 module**:`algo/go.mod` 一个 module,每题 `go/` 子目录是独立 package,`go test ./...` 一把梭。

## 关联

- 学习方法 + 题单 + 4 周逐日落地计划:见会话 outputs 的 `Algo-DS-4Week-Sprint.md` 与 `落地执行计划.md`
- ultragoal 目标:`goals/algo-interview-prep/`
