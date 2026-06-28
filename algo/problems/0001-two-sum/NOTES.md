# Two Sum(LC 1)· 五语言对比笔记

## 核心思路(语言无关)

一次遍历 + 哈希表。遍历到 `x` 时,查表里有没有 `target - x`;有就返回两个下标,没有就把 `x → 下标` 存进表。**时间 O(n)、空间 O(n)**。关键洞察:用"O(1) 查找"把暴力的内层循环消掉。

## 五语言关键差异

| 维度 | Python | Go | JavaScript | Java | Scala |
|---|---|---|---|---|---|
| 哈希表类型 | `dict` | `map[int]int` | `Map`(非 `{}`) | `HashMap<Integer,Integer>` | `mutable.HashMap` |
| 存在性判断 | `need in seen` | `v, ok := m[k]` 双返回值 | `seen.has(need)` | `containsKey` | `seen.get(k)` → `Option` |
| 早返回惯用 | `return [...]` | `return []int{...}` | `return [...]` | `return new int[]{...}` | 避免 `return`,用标志位 |
| "空"返回 | `[]` | `nil` | `[]` | `new int[0]` | `Array.empty[Int]` |

### 各语言要点

- **Python**:`dict` 即哈希表,`in` 走 `__hash__`。最 Pythonic。也可用海象运算符:`if (need := target - x) in seen:`。
- **Go**:map 的 **comma-ok**(`j, ok := seen[target-x]`)是判断 key 是否存在的标准姿势——不能只看零值,因为下标 0 是合法值。`make(map[int]int, len(nums))` 预设容量减少 rehash。
- **JavaScript**:**务必用 `Map` 而不是普通对象 `{}`**。对象的 key 会被转成字符串、还会撞到原型链上的 `toString` 等;`Map` 保留键类型、有真正的 `.has()`、迭代有序。这是 JS 里最容易踩的坑。
- **Java**:装箱是隐藏成本——`HashMap<Integer,Integer>` 的 key/value 都是包装类,有自动装箱/拆箱开销。`map.get` 返回 `Integer` 可能为 `null`,所以先 `containsKey`。
- **Scala**:`seen.get(k)` 返回 `Option[Int]`,配 `match`/`getOrElse` 是惯用法,从类型层面消除了"key 不存在"的 NPE。Scala 文化里**尽量不用 `return`**(它会抛 `NonLocalReturnControl`),所以本实现用 `while + 标志位` 提前结束。纯函数式可用 `nums.zipWithIndex.foldLeft(...)`,但可读性变差,这里取务实写法。

## 最近实践 / 进阶

- **返回顺序**:存的是"先出现的下标",所以返回 `[seen[need], i]`(前者在前),符合题目期望。
- **重复值**:`[3,3]` 能正确返回 `[0,1]`,因为查表在写表之前。
- **JS 性能**:`Map` 在数值键场景比 object 快且无坑;别用 `obj[key]`。
- **Go 进阶**:若值类型只为"存在性"用 `map[int]struct{}`(零内存 value),见 Contains Duplicate。
- **Scala 进阶**:`collection.mutable.Map` vs 不可变 `Map`——刷题用 mutable 更直接;生产代码偏好不可变 + 函数式。

## 复杂度

| | 时间 | 空间 |
|---|---|---|
| 本解法 | O(n) | O(n) |
| 暴力双循环 | O(n²) | O(1) |

## 易错点

1. JS 用了 `{}` 而非 `Map` → 数值键被转字符串,虽然本题恰好能过,但属坏习惯。
2. Go 只写 `if seen[target-x] != 0` → 下标 0 被误判,必须用 comma-ok。
3. 先写表后查表 → 自己和自己配对。务必"先查后写"。
