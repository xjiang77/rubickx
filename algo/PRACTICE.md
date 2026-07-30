# 练习模式（Go / Java 手写）

每道题在原有参考解之外，多了一份**留空的练习**：签名、导入、思路提示、示例测试都给好了，
核心算法留给你自己写。目前覆盖 Go 和 Java 两种语言。

## 文件长什么样

以 `problems/0001-two-sum/` 为例：

```
go/
  twosum.go                    参考解（别动）
  twosum_test.go               参考解的测试
  twosum_practice.go           <- 你写这里：func TwoSumPractice(...)
  twosum_practice_test.go      <- 你写这里：一个示例用例 + 一堆 TODO
java/
  TwoSum.java                  参考解（别动）
  TwoSumTest.java              参考解的测试
  TwoSumPractice.java          <- 你写这里：class TwoSumPractice
  TwoSumPracticeTest.java      <- 你写这里：一个 @Test 示例 + 一堆 TODO
```

命名规则很死板，方便肌肉记忆：Go 是函数名加 `Practice` 后缀、文件名加 `_practice`；
Java 是类名加 `Practice` 后缀。Java 用的是默认包、全仓库一个 classpath，所以类名必须全局唯一，
`Practice` 后缀刚好也解决了这个问题。

## 怎么跑

```bash
make practice          # Go + Java 的练习一起跑
make practice-go       # 只跑 Go 练习
make practice-java     # 只跑 Java 练习

# 只跑单道题的 Go 练习
go test -tags practice ./problems/0001-two-sum/go/ -v
```

**刚拿到手时 `make practice` 一定是全红的** —— 8 道题 8 个失败，因为实现全是 `TODO`。
这就是起点，红转绿的过程就是练习本身。

## 为什么 `make test` 不会被练习拖红

- **Go**：练习文件顶部有 `//go:build practice`，默认的 `go test ./...` 根本不会编译它们。
- **Java**：练习测试类上有 `@Tag("practice")`，`make test-java` 用 `--exclude-tag=practice` 跳过，
  `make practice-java` 用 `--include-tag=practice` 只跑它们。
  （注意 Java 这边所有源文件仍然会被 `javac` 编译，所以练习实现即使没写完也必须能编译通过 ——
  骨架里已经放了 `return new int[0];` 这类占位返回值，别删掉返回语句。）

所以 `make test` 永远只反映参考解的状态，`make practice` 只反映你自己的状态，两条线互不干扰。

## 建议的练习节奏

1. 先看 `NOTES.md` 里的题意和复杂度目标，**不要**看参考解。
2. 打开练习文件，只读顶部注释里的「思路提示」，能不看就不看。
3. 写实现，`make practice-go` 跑到示例用例过。
4. **回到练习测试文件，把 TODO 里列的场景补成真正的用例**。这一步比写实现更值钱 ——
   面试里区分度最高的往往不是你能不能写出主逻辑，而是你有没有主动说出边界情况。
5. 补完的用例如果把自己的实现打红了，说明第 3 步漏了东西，回去修。
6. 全绿之后再对着参考解 diff，看语言惯用法上的差异（Go 的 `map[T]struct{}` vs Java 的 `HashSet`，等等）。
7. 隔几天把练习文件清空重来一遍，这才是真正的「刷」。

## 各题的提示强度

练习文件顶部注释按「越往下越具体」排列，卡住了从上往下一条一条看，看到能动手就停：

| 题 | 提示里给到的最深一层 |
|---|---|
| 1 Two Sum | 先查后存，否则 `target = 2x` 时会自己配自己 |
| 217 Contains Duplicate | Java 的 `HashSet.add` 返回值本身就是判重信号 |
| 242 Valid Anagram | Go 的 `len(s)` 是字节数，多字节字符要按 rune 走 |
| 49 Group Anagrams | key 可以是排序后的串，也可以是 26 位计数签名 |
| 347 Top K Frequent | 桶下标 = 出现次数，倒着扫桶收 k 个 |
| 238 Product Except Self | 第二遍用滚动变量累乘后缀积，做到 O(1) 额外空间 |
| 36 Valid Sudoku | 宫下标 `b = (r/3)*3 + c/3` |
| 128 Longest Consecutive | 只从 `x-1` 不在集合里的起点展开，否则退化成 O(n²) |

## 加新题时

新增一道题后，照着现有骨架补四个文件即可，注意三件事：

1. Go 练习文件和练习测试文件**都要**带 `//go:build practice`，漏一个就会编译报错（函数找不到 / 重复声明）。
2. Java 练习测试类**必须**带 `@Tag("practice")`，否则会混进 `make test`。
3. Go 练习测试里的辅助函数（`toBoard`、`normalize` 这类）不能和参考解测试里的重名 ——
   它们在同一个 package 下，现有骨架统一加了 `Practice` 后缀。
