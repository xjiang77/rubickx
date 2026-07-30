# Longest Consecutive Sequence(LC 128)· 五语言对比笔记

## 核心思路(语言无关)

要求 O(n),排序(O(n log n))出局。做法:

1. 全部丢进 **set**(去重 + O(1) 查询)。
2. 只对**序列起点**展开计数——`x` 是起点 ⇔ `x-1` 不在集合里。
3. 从起点向右 `x+1, x+2, ...` 逐个查,数出长度。

**为什么是 O(n)**:非起点直接跳过(O(1));每个元素只会被它所在序列的唯一起点向右扫到一次。总访问 ≤ 2n,均摊 O(n)。面试必讲清这一步,否则看起来像 O(n²)。

## 五语言关键差异

| 维度 | Python | Go | JavaScript | Java | Scala |
|---|---|---|---|---|---|
| set 类型 | `set(nums)` | `map[int]struct{}` | `new Set(nums)` | `HashSet<Integer>` | `nums.toSet`(不可变) |
| 成员查询 | `x in s` | comma-ok | `set.has(x)` | `contains` | `set(x)` 直接当函数 |
| 空输入处理 | `best = 0` 起步 | 同 | 同 | 同 | **`maxOption.getOrElse(0)`** |

### 各语言要点

- **Python**:`set(nums)` 一步去重;`while x + length in s` 直接读作数学谓词。最短最清晰的版本。
- **Go**:**`map[int]struct{}` 是 Go 的标准 set 惯用法**——`struct{}` 零字节,比 `map[int]bool` 省 value 存储(bool 版胜在 `if set[x]` 可直接判断,二选一均可,但要能说出差异)。Contains Duplicate 笔记里埋的伏笔在这题兑现。
- **JavaScript**:`new Set(nums)` 构造即去重;**遍历 `set` 而不是原数组 `nums`**——数组含大量重复时能省一轮(五语言同理,Python/Go/Java 版同样遍历 set)。
- **Java**:`HashSet` + 增强 for。装箱开销依旧(`Set<Integer>`),数据量大时 `contains` 的常数比 Go/JS 高;面试可提"可用 IntOpenHashSet(fastutil)消装箱",体现工程敏感度。
- **Scala**:三个惯用法一次示范——① `Set[Int]` 本身就是 `Int => Boolean` 函数,`takeWhile(set)` 直接把集合当谓词传;② **`Iterator.from(x)` 惰性无穷序列**,`takeWhile` 截断,不会真的无穷跑;③ `maxOption.getOrElse(0)` 类型安全地处理空集(直接 `.max` 空集会抛异常)。整条链无可变状态。

## 复杂度

| 方案 | 时间 | 空间 |
|---|---|---|
| set + 起点展开(本解) | 均摊 O(n) | O(n) |
| 排序后线性扫 | O(n log n) | O(1)~O(n) |

排序版虽然复杂度更差,但常数小、代码也短,输入巨大且内存紧张时反而实用——能主动谈这个取舍是加分项。

## 易错点

1. 忘了"只从起点展开"的判断 → 每个元素都向右扫,最坏 O(n²)(全连续时),面试当场超时。
2. 对**原数组**每个元素展开而不是对 set → 大量重复元素时重复扫描;先去重再遍历。
3. 空数组:`max` 没有初值会崩(Scala `.max` 抛 UnsupportedOperationException,Python `max([])` 抛 ValueError)——`best = 0` 起步或 `maxOption`。
4. 重复元素(如两个 0)不影响正确性(set 已去重),但测试要覆盖——LC 官方样例 2 特意放了重复。
