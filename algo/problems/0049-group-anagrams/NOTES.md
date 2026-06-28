# Group Anagrams(LC 49)· 五语言对比笔记

## 核心思路

把互为异位词的字符串分到一组。关键是给每个字符串算一个**规范化的 key**,让同组字符串 key 相同:
- **key = 排序后的字符串**(如 `"eat"`→`"aet"`):简单,O(k log k) 每个字符串。
- **key = 26 维字符计数签名**(如 `"a1e1t1"`):O(k) 每个字符串,k 大时更优。

然后用哈希表 `key → 该组列表` 聚合。本仓库用"排序 key"。整体 O(n·k·log k)。

## 五语言"分组聚合"对比

| 语言 | 算 key | 聚合惯用法 |
|---|---|---|
| Python | `"".join(sorted(s))` | `defaultdict(list)`,`groups[key].append(s)` |
| Go | `sort.Slice([]byte(s))` | `map[string][]string`,`append` 自动建组 |
| JavaScript | `s.split('').sort().join('')` | `Map`,先 `has` 再 `push` |
| Java | `Arrays.sort(char[])` | `computeIfAbsent(key, k -> new ArrayList<>())` |
| Scala | `s.sorted` | `strs.groupBy(_.sorted)` 一行 |

### 要点

- **Scala 完胜简洁度**:`strs.groupBy(s => s.sorted).values.map(_.toList).toList`。`groupBy` 是标准库一等公民,`String.sorted` 直接返回排好序的字符串。函数式表达"按 key 分组"的天花板。
- **Java 的 `computeIfAbsent`**(Java 8+):`map.computeIfAbsent(key, k -> new ArrayList<>()).add(s)` 一步完成"没有就建空列表再添加",取代了 `if (!map.containsKey) map.put(...)` 的样板。这是现代 Java 的标志写法。
- **Go**:`map` 的 `append(groups[key], s)` 利用了"对 nil slice append 也合法"——key 不存在时 `groups[key]` 是 `nil`,append 后再赋回,自动建组,无需判空。排序字符要先 `[]byte(s)` 再 `sort.Slice`(字符串不可变)。
- **Python 的 `defaultdict(list)`**:访问缺失 key 自动建空列表,省去 `setdefault`。最 Pythonic。
- **JS** 没有 `defaultdict`/`computeIfAbsent`,得手动 `if (!groups.has(key)) groups.set(key, [])`,略啰嗦。`[...groups.values()]` 用扩展运算符把迭代器收成数组。

## 结果顺序与测试

分组的**组间顺序、组内顺序都不保证**(取决于哈希迭代序)。所以 5 份测试都先做**归一化**(组内排序 + 组间按拼接串排序)再比较,而不是直接 `equals`。这是写"集合/分组类"题测试的通用技巧,值得记住。

## 进阶:更优的 key

用字符计数签名(长度 26 的计数数组转成字符串)做 key,可把每个字符串的处理从 O(k log k) 降到 O(k):
- Java:`int[26]` → `Arrays.toString` 或自定义编码
- Python:`tuple(counts)` 直接当 dict key(元组可哈希)
- Go:把 `[26]int` 数组(可比较、可作 map key)直接当 key
- 面试加分项:被问"还能更快吗"时给出这个 O(n·k) 方案。

## 复杂度

| key 策略 | 单串 | 总体 |
|---|---|---|
| 排序 key | O(k log k) | O(n·k·log k) |
| 计数签名 key | O(k) | O(n·k) |

n=字符串个数,k=字符串平均长度。
