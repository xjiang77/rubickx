// Package hashmap — W1 kata #2:手写链地址法 HashMap。
//
// 目标:理解哈希表三件事——哈希函数、碰撞处理、扩容 rehash。
// 禁止内部使用 built-in map;哈希函数自己实现 FNV-1a。
//
// 完成标准:go test ./structures/go/hashmap/ 全绿;
// 然后思考题:runtime/map.go 用的是开放寻址还是链地址?tophash 是干什么的?
// (读 src/runtime/map.go 前 100 行找答案,W3 会回收这个问题)
//
// 契约细节见 hashmap_test.go 顶部注释。
package hashmap

// entry 是链表节点。
type entry[V any] struct {
	key  string
	val  V
	next *entry[V]
}

// Map 是 string → V 的链地址法哈希表。
type Map[V any] struct {
	buckets []*entry[V]
	n       int // 元素个数
}

// New 创建默认 8 个桶的 Map。
func New[V any]() *Map[V] { return NewWithCapacity[V](8) }

// NewWithCapacity 创建指定桶数的 Map(至少 1 个桶)。
func NewWithCapacity[V any](buckets int) *Map[V] {
	if buckets < 1 {
		buckets = 1
	}
	return &Map[V]{buckets: make([]*entry[V], buckets)}
}

// fnv1a 是 FNV-1a 64-bit 哈希。
// 常量:offset basis = 14695981039346656037,prime = 1099511628211。
func fnv1a(s string) uint64 {
	panic("TODO: implement fnv1a (对每个字节:先 XOR 再乘 prime)")
}

// Len 返回元素个数。
func (m *Map[V]) Len() int { return m.n }

// Buckets 返回桶数(测试用它验证负载因子)。
func (m *Map[V]) Buckets() int { return len(m.buckets) }

// Put 插入或覆盖。插入前若 (n+1)/buckets > 0.75 则先 2x 扩容。
func (m *Map[V]) Put(key string, val V) {
	panic("TODO: implement Put (三步:检查负载因子 → 找链上同 key 覆盖 → 头插新节点)")
}

// Get 查找;第二返回值表示 key 是否存在。
func (m *Map[V]) Get(key string) (V, bool) {
	panic("TODO: implement Get")
}

// Delete 删除;返回是否真的删除了元素。
// 注意链表删除的两种情况:头节点 vs 中间节点。
func (m *Map[V]) Delete(key string) bool {
	panic("TODO: implement Delete")
}

// Range 遍历全部键值对;回调返回 false 时提前终止。
func (m *Map[V]) Range(f func(key string, val V) bool) {
	panic("TODO: implement Range")
}

// resize 桶数翻倍并把所有 entry rehash 到新桶。
func (m *Map[V]) resize() {
	panic("TODO: implement resize (思考:为什么必须重算每个 key 的桶下标?)")
}
