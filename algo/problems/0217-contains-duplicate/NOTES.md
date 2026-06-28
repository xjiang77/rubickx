# Contains Duplicate(LC 217)· 五语言对比笔记

## 核心思路

遍历,用**集合(Set)**记录见过的值;再次遇到已存在的即返回 `true`。边遍历边查可提前退出。时间 O(n)、空间 O(n)。

## 五语言的"集合"长什么样

| 语言 | 集合类型 | 添加并判重的惯用法 |
|---|---|---|
| Python | `set` | `if x in seen: ... ; seen.add(x)` |
| Go | `map[int]struct{}` | 无内建 set,用 map + 空结构体 value |
| JavaScript | `Set` | `seen.has(x)` / `seen.add(x)` |
| Java | `HashSet<Integer>` | `if (!seen.add(x)) return true;` |
| Scala | `mutable.HashSet` | `if !seen.add(x) then ...` |

### 要点

- **Go 没有内建 Set**:惯用 `map[T]struct{}`,`struct{}{}` 不占内存(0 字节),比 `map[T]bool` 更省。这是 Go 的标志性写法。
- **Java / Scala 的 `add` 返回布尔**:`Set.add` 返回"是否真的新增了"。`!seen.add(x)` 为 `true` 说明已存在——一行完成判重,无需先 `contains` 再 `add`(少一次哈希)。
- **Python / JS** 没有这个返回值便利,得 `in`/`has` 后再 `add`(两次哈希,但代码直观)。

## 各语言"一行版"(牺牲提前退出)

- Python:`return len(set(nums)) != len(nums)`
- JS:`return new Set(nums).size !== nums.length`
- Java(Stream):`return nums.length != Arrays.stream(nums).distinct().count();`
- Scala:`nums.toSet.size != nums.length` 或 `nums.distinct.length != nums.length`
- Go:无优雅一行,仍需显式循环

> 取舍:一行版要先建整个集合(无法在发现第一个重复时就停),最坏/平均都 O(n) 空间且常数更大;面试里**显式循环 + 提前 return** 更体现你懂复杂度。

## 复杂度

时间 O(n)、空间 O(n)。若允许改原数组:排序后比较相邻,O(n log n) 时间、O(1) 额外空间——空间换时间的另一选择,值得在面试里提一句。

## 易错点

- Go 用 `map[int]bool` 也行,但 `map[int]struct{}` 更地道、更省内存。
- Scala/Java 别忘了 `add` 的布尔返回值能省一次哈希。
