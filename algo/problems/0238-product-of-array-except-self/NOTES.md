# Product of Array Except Self(LC 238)· 五语言对比笔记

## 核心思路(语言无关)

`res[i] = (i 左边所有数的积) × (i 右边所有数的积)`。两趟遍历:

1. 正向:`res[i]` 先存**前缀积**(不含自己)。
2. 反向:用一个滚动变量 `suffix` 累计**后缀积**,原地乘进 `res[i]`。

**为什么不用除法**:题目明说禁用;更本质的原因是数组含 0 时"总积 ÷ 自己"直接崩(除零 + 信息丢失)。前缀/后缀分解是**扫描线思想**的入门样板,后面 Trapping Rain Water 等题同款。

## 五语言关键差异

| 维度 | Python | Go | JavaScript | Java | Scala |
|---|---|---|---|---|---|
| 结果数组初始化 | `[1] * n` | `make([]int, n)` | `new Array(n).fill(1)` | `new int[n]` | `Array.tabulate` |
| 反向遍历 | `range(n-1, -1, -1)` | 经典 `for i--` | 经典 `for i--` | 经典 `for i--` | 无需(scan 代替) |
| 前缀积 API | 手写(`itertools.accumulate` 可选) | 手写 | 手写 | 手写 | **`scanLeft` 内建** |

### 各语言要点

- **Python**:`range(n - 1, -1, -1)` 的三参形式是标准反向遍历;`reversed(range(n))` 更易读,任选。进阶:`itertools.accumulate(nums, mul)` 就是前缀积,但两趟手写更清楚。
- **Go**:`make([]int, n)` 天生全 0,注意本题要的初值语义是"乘法单位元 1",第一趟赋值 `res[i] = prefix` 恰好覆盖了零值——如果先乘后赋就错了,零值哲学这里反而要小心。
- **JavaScript**:两个坑。① `new Array(n)` 是**空洞数组**(holes),直接 `map` 会跳过空洞——必须 `.fill(1)`(或 `Array.from`)。② JS 数字是 float64,**负数乘 0 会得到 `-0`**(本题 `[-1,1,0,-3,3]` 的结果里就有):`-0 === 0` 为 true,但 `Object.is(-0, 0)` 为 false,jest 的 `toEqual` 按后者比较——测试里用 `x + 0` 归一化(`-0 + 0 === 0`)。其他四语言用整数,无此问题。
- **Java**:原始类型 `int[]` 无装箱开销,这题是 Java 最舒服的形态。溢出提示:LC 保证乘积在 32 位内,真实场景要考虑 `long` 或 `Math.multiplyExact`(溢出抛异常而非静默回绕)。
- **Scala**:`scanLeft(1)(_ * _)` / `scanRight(1)(_ * _)` 直接生成前缀/后缀积数组(比输入长 1,含初始单位元),`res(i) = prefix(i) * suffix(i+1)` 一行组合——**scan 是"前缀聚合"的标准库抽象**,值得单独记住。代价是两个 O(n) 中间数组;要 O(1) 额外空间就照 Python 版命令式翻译。

## 复杂度

| 方案 | 时间 | 额外空间(不含输出) |
|---|---|---|
| 前缀×后缀,滚动变量(Py/Go/JS/Java) | O(n) | **O(1)** |
| scanLeft + scanRight(Scala 版) | O(n) | O(n) |
| 除法方案(反例) | O(n) | O(1),但含 0 即错 |

## 易错点

1. JS `new Array(n)` 不 `fill` 就 `map`/迭代 → 空洞被跳过,结果全是 `undefined`。
2. JS 浮点 `-0`:`1 * 0 * -3 * 3` 得 `-0`,jest `toEqual` 视 `-0 ≠ 0`(Object.is 语义)→ 断言前 `+ 0` 归一化。
3. 第二趟先 `suffix *= nums[i]` 再乘进 res → 把自己也乘进去了。顺序必须:**先用后更新**。
4. 用除法 + 遇 0 → 除零错误;就算特判 0 也要分"一个 0 / 两个 0"两种情况,复杂度暴涨,面试直接说"不用除法"。
5. Scala `scanLeft` 结果长度是 n+1(带初始值),下标别对错位:`prefix(i)` 是前 i 个数的积,`suffix(i+1)` 是 i 之后的积。
