# structures —— Go Sprint 手写数据结构 kata

配套 vault:`[[05 - Go Sprint 8-Week Plan]]`(线 B)。每个结构一个目录:**测试是完备的契约,实现留白**。红 → 绿 → 重构,然后回答目录内代码注释里的思考题(它们会连回 golang/go 源码)。

规则:

- 禁止用标准库同类容器偷懒(动态数组不用 built-in `append`,hashmap 内部不用 built-in `map`)。
- 全绿后跑 `go test -bench .`,把关键 benchmark 数字记进 NOTES 或 vault Session Log。
- 完成状态以本目录测试为准,PROGRESS.md 只记 ✅/⬜。

## Kata 清单(按周解锁)

| 周 | 结构 | 目录 | 状态 |
|----|------|------|------|
| W1 | 动态数组 | `go/dynamicarray/` | ⬜ |
| W1 | 链地址法 HashMap(FNV-1a) | `go/hashmap/` | ⬜ |
| W2 | 双端队列(环形缓冲) | `go/deque/`(W2 生成) | — |
| W2 | LRU(双向链表 + map) | `go/lru/` | — |
| W3 | 二叉堆 + container/heap 适配 | `go/heap/` | — |
| W4 | Trie | `go/trie/` | — |
| W4 | 并查集(路径压缩 + 按秩合并) | `go/unionfind/` | — |
| W5 | 图(邻接表 + BFS/DFS/Dijkstra) | `go/graph/` | — |
| W8 | 跳表(stretch) | `go/skiplist/` | — |

跑法(在 `algo/` 下):

```bash
go test ./structures/...          # 全部 kata
go test -bench . ./structures/go/dynamicarray/
```

后续每周的 kata 由 go-sprint-coach session 生成(同样是测试完备 + 实现留白)。
