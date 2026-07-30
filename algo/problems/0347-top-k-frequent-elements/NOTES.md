# Top K Frequent Elements(LC 347)· 五语言对比笔记

## 核心思路(语言无关)

两步:**计数** → **按频次取前 k**。取前 k 有三档方案:

1. **排序**:按频次降序排,取前 k。O(n log n),最好写。
2. **堆**:维护大小为 k 的最小堆。O(n log k),k 远小于 n 时优。
3. **桶排序**:频次最大只能是 n,开 n+1 个桶,下标=频次,从高往低收集。**O(n)**,本仓库主解法。

桶排序能到 O(n) 的洞察:**频次的值域天然有界(1..n)**,不需要比较排序。

## 五语言关键差异

| 维度 | Python | Go | JavaScript | Java | Scala |
|---|---|---|---|---|---|
| 计数惯用法 | `Counter(nums)` | `counts[x]++` | `counts.set(x, (counts.get(x) ?? 0) + 1)` | `counts.merge(x, 1, Integer::sum)` | `groupMapReduce(identity)(_ => 1)(_ + _)` |
| 桶结构 | `list[list[int]]` | `[][]int` + `append` | `Array.from({length}, () => [])` | `List<List<Integer>>` | (用排序替代) |
| 一步到位 API | `Counter.most_common(k)` | 无 | 无 | 无 | `sortBy(-_._2).take(k)` |

### 各语言要点

- **Python**:`Counter` 就是为这题生的——`Counter(nums).most_common(k)` 一行出答案(内部是 heapq,O(n log k))。手写桶排序为了讲清 O(n) 原理;面试先报 `most_common`,追问"更快?"再给桶。
- **Go**:map 计数 `counts[x]++` 直接可写(缺失 key 返回零值,无需判存在)——这是 Go 零值哲学的红利。`append(buckets[c], x)` 对 nil slice 同样合法。注意收集时可能一桶多个导致超 k,末尾 `res[:k]` 截断。
- **JavaScript**:`??`(nullish coalescing)是现代计数姿势,比 `|| 0` 安全(`|| 0` 会把合法的 0 也覆盖,计数场景恰好无害但习惯要对)。**`sort()` 默认按字符串比较**,数字必须传 `(a, b) => a - b`——测试里专门示范。
- **Java**:`merge(x, 1, Integer::sum)` 是现代计数标准写法,取代 `getOrDefault` + `put` 两步。桶不能用 `new ArrayList[n]`(泛型数组),用 `List<List<Integer>>` 预填充。
- **Scala**:`groupMapReduce(键)(映射)(归约)` 一个调用完成 groupBy + mapValues + reduce,是 2.13 加入的"计数/聚合"终极 API。整个解法是一条无副作用的管道,表达力五语言最强;代价是 O(n log n) 排序——NOTES 明确记录这个取舍,追求 O(n) 时照 Python 版翻译桶排序即可。

## 结果顺序与测试

题目不要求输出顺序,桶收集顺序取决于 map 迭代序(Go 甚至刻意随机化)。所以**五份测试都先排序再比较**——和 Group Anagrams 的归一化技巧同源:输出无序的题,测试先归一化。

## 复杂度

| 方案 | 时间 | 空间 |
|---|---|---|
| 桶排序(Py/Go/JS/Java) | O(n) | O(n) |
| 排序(Scala 版) | O(n log n) | O(n) |
| 最小堆 | O(n log k) | O(n + k) |

## 易错点

1. JS `sort()` 不传比较器 → `[10, 2]` 排成 `[10, 2]`(字典序)。数字排序永远写 `(a, b) => a - b`。
2. 桶下标从高往低扫时,同一桶元素可能把结果推过 k → 记得截断(Go `res[:k]`、JS `slice(0, k)`、Java 内层 `return`)。
3. 桶要开 **n+1** 个:全部元素相同时频次=n,`buckets[n]` 必须存在。
4. Java 写 `new List<Integer>[n]` → 泛型数组编译错误,老老实实 `List<List<>>`。
