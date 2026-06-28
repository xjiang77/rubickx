# Valid Anagram(LC 242)· 五语言对比笔记

## 核心思路

互为字母异位词 ⇔ **每个字符出现次数相同**。两条主流写法:
1. **计数**:统计 s 的字符频次,再用 t 抵消;任一变负或最后非零即否。O(n) 时间、O(k) 空间(k=字符集)。
2. **排序后比较**:`sorted(s) == sorted(t)`。O(n log n),代码短但更慢。

本仓库 5 种实现都用"计数",但**计数容器的选择**体现了语言差异。

## 五语言计数方式对比

| 语言 | 计数容器 | 特点 |
|---|---|---|
| Python | `collections.Counter` | 标准库直接给"多重集",`Counter(s) == Counter(t)` 一行 |
| Go | `map[rune]int` | 按 `rune` 遍历正确处理多字节;`len` 是字节数,仅作快速剪枝 |
| JavaScript | `Map` | `count.get(c) ?? 0` 处理未定义;`!n` 同时挡住 `undefined` 和 `0` |
| Java | `int[26]` 定长数组 | 假定小写字母,`c - 'a'` 当下标,O(1) 空间、最快 |
| Scala | `groupMapReduce` | `str.groupMapReduce(identity)(_ => 1)(_ + _)` 一次遍历出频次表 |

### 要点

- **Java 的 `int[26]`** 是最"贴硬件"的写法:数组下标计数,没有装箱、没有哈希,常数最小。代价是隐含假设(仅小写字母 a–z)。面试中要主动说出这个前提,并说明 Unicode 场景应改用 `HashMap`。
- **Go 的 byte vs rune**:`len(s)` 返回字节数;`for _, c := range s` 按 `rune`(Unicode 码点)遍历。两者在 ASCII 下一致,在中文/emoji 下不同。这是 Go 字符串最经典的考点。
- **Scala 的 `groupMapReduce`**(2.13+):一次遍历完成 group + map + reduce,等价于"分组计数",是函数式表达频次表的现代最佳实践,比 `groupBy(identity).view.mapValues(_.size)` 更省一次中间集合。
- **Python 的 `Counter`** 直接支持 `==` 比较多重集,最简。先比 `len` 是廉价剪枝。
- **JS 的 `?? 0` 与 `!n`**:`??`(空值合并)只在 `null/undefined` 时取默认,比 `||` 更准(`0 || x` 会错误地落到 x)。`if (!n)` 巧妙地用一个判断同时拦住"字符不存在(undefined)"和"已抵消完(0)"。

## 复杂度

| 写法 | 时间 | 空间 |
|---|---|---|
| 计数(数组/map) | O(n) | O(k) / O(1) |
| 排序比较 | O(n log n) | O(n)(多数语言 sort 字符串) |

## 易错点 / Follow-up

- **Unicode**:`int[26]` 在含非 a–z 字符时越界或漏算;改 `Map<Character,Integer>` 或按码点计数。
- **大小写 / 空格**:题目通常限定小写,若不限要先归一化。
- 面试官常追问:"如果是 Unicode 怎么改?" —— 答:从定长数组换成哈希表,Go 已天然按 rune。
