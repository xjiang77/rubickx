# 进度追踪

✅ = 解法 + 测试完成且通过  ·  ⬜ = 未做  ·  🟡 = 解法已写未验证(Scala 待本机跑)

| # | 题目 | Pattern | Py | Go | JS | Java | Scala | NOTES |
|---|---|---|---|---|---|---|---|---|
| 1 | Two Sum | 哈希 | ✅ | ✅ | ✅ | ✅ | 🟡 | ✅ |
| 217 | Contains Duplicate | 哈希/集合 | ✅ | ✅ | ✅ | ✅ | 🟡 | ✅ |
| 242 | Valid Anagram | 计数 | ✅ | ✅ | ✅ | ✅ | 🟡 | ✅ |
| 49 | Group Anagrams | 哈希分组 | ✅ | ✅ | ✅ | ✅ | 🟡 | ✅ |
| 347 | Top K Frequent | 计数+桶排序 | ✅ | ✅ | ✅ | ✅ | 🟡 | ✅ |
| 238 | Product of Array Except Self | 前缀/后缀积 | ✅ | ✅ | ✅ | ✅ | 🟡 | ✅ |
| 36 | Valid Sudoku | 集合/位掩码 | ✅ | ✅ | ✅ | ✅ | 🟡 | ✅ |
| 128 | Longest Consecutive Sequence | 集合+起点展开 | ✅ | ✅ | ✅ | ✅ | 🟡 | ✅ |

> Day 1(6/29)+ Day 2(7/21)已落地,**哈希/数组专题(NeetCode Arrays & Hashing 8 题)完结**。
> Python / JS / Go / Java 均已在沙箱跑通(Java 用 JUnit 5 console,沙箱里 17 tests 全绿);Scala 待本机 `make test-scala` 验证后把 🟡 改 ✅。
> 沙箱验证备注:Day 2 会话里 `go test ./...` 与 JUnit 把 Day 1 的 Go/Java 也一并验证了,故其 🟡 已转 ✅。

## 接下来(按落地执行计划)

- Day 3:双指针 —— Valid Palindrome(125)、Two Sum II(167)、3Sum(15)、Container With Most Water(11)
- Day 4:滑动窗口 —— Best Time to Buy and Sell Stock(121)、Longest Substring Without Repeating(3)、Longest Repeating Character Replacement(424)
- …详见 `落地执行计划.md`
