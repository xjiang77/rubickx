// Package dynamicarray — W1 kata #1:手写动态数组。
//
// 目标:理解 Go slice 背后的机制(backing array、len/cap、扩容拷贝),
// 所以这里禁止用 built-in append 偷懒 —— 自己管理 backing slice 与逻辑长度。
//
// 完成标准:go test ./structures/go/dynamicarray/ 全绿,
// 然后 go test -bench . 观察均摊 O(1),并和直接用 append 的版本对比。
//
// 契约细节见 dynamicarray_test.go 顶部注释。
package dynamicarray

// Array 是一个泛型动态数组。
// data 是 backing slice(len(data) 即容量),n 是逻辑长度。
type Array[T any] struct {
	data []T
	n    int
}

// New 创建空数组。
func New[T any]() *Array[T] {
	return &Array[T]{}
}

// Len 返回逻辑长度。
func (a *Array[T]) Len() int {
	return a.n
}

// Cap 返回当前容量。
func (a *Array[T]) Cap() int {
	return len(a.data)
}

// Append 在尾部追加,容量不足时 2x 扩容(空数组扩到至少 1)。
func (a *Array[T]) Append(v T) {
	panic("TODO: implement Append (先写 grow,再写追加)")
}

// Get 返回下标 i 的元素;越界 panic。
func (a *Array[T]) Get(i int) T {
	panic("TODO: implement Get")
}

// Set 覆写下标 i 的元素;越界 panic。
func (a *Array[T]) Set(i int, v T) {
	panic("TODO: implement Set")
}

// Insert 在下标 i 处插入(允许 i == Len(),等价 Append);其后元素右移。
func (a *Array[T]) Insert(i int, v T) {
	panic("TODO: implement Insert (注意 copy 的方向与区间)")
}

// RemoveAt 删除并返回下标 i 的元素;其后元素左移。
// 思考:为什么删除后要把最后一个 slot 置零值?(提示:GC 与内存泄漏)
func (a *Array[T]) RemoveAt(i int) T {
	panic("TODO: implement RemoveAt")
}
