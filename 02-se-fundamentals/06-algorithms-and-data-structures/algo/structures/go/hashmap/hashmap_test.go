package hashmap

import (
	"fmt"
	"testing"
)

// 契约:
//   链地址法(separate chaining)HashMap[string]V,自实现 FNV-1a 哈希,禁止内部用 built-in map
//   Put 覆盖同 key;Get 第二返回值表示是否存在;Delete 返回是否删除了元素
//   负载因子 > 0.75 时 2x 扩容并 rehash
//   NewWithCapacity(n) 允许指定初始桶数,便于测试碰撞路径

func TestPutGet(t *testing.T) {
	m := New[int]()
	m.Put("a", 1)
	m.Put("b", 2)
	if v, ok := m.Get("a"); !ok || v != 1 {
		t.Errorf("Get(a) = %d,%v want 1,true", v, ok)
	}
	if v, ok := m.Get("b"); !ok || v != 2 {
		t.Errorf("Get(b) = %d,%v want 2,true", v, ok)
	}
	if _, ok := m.Get("missing"); ok {
		t.Error("Get(missing) ok = true, want false")
	}
}

func TestOverwrite(t *testing.T) {
	m := New[string]()
	m.Put("k", "v1")
	m.Put("k", "v2")
	if v, _ := m.Get("k"); v != "v2" {
		t.Errorf("Get(k) = %q, want v2", v)
	}
	if m.Len() != 1 {
		t.Errorf("Len() = %d, want 1 (overwrite must not grow)", m.Len())
	}
}

func TestDelete(t *testing.T) {
	m := New[int]()
	m.Put("x", 1)
	if !m.Delete("x") {
		t.Error("Delete(x) = false, want true")
	}
	if m.Delete("x") {
		t.Error("second Delete(x) = true, want false")
	}
	if _, ok := m.Get("x"); ok {
		t.Error("Get(x) after delete: ok = true")
	}
	if m.Len() != 0 {
		t.Errorf("Len() = %d, want 0", m.Len())
	}
}

func TestCollisionChaining(t *testing.T) {
	// 1 个桶:所有 key 都碰撞,链表必须仍然正确
	m := NewWithCapacity[int](1)
	for i := 0; i < 50; i++ {
		m.Put(fmt.Sprintf("key%d", i), i)
	}
	for i := 0; i < 50; i++ {
		if v, ok := m.Get(fmt.Sprintf("key%d", i)); !ok || v != i {
			t.Fatalf("Get(key%d) = %d,%v want %d,true", i, v, ok, i)
		}
	}
	// 链中间删除
	if !m.Delete("key25") {
		t.Fatal("Delete(key25) failed")
	}
	if _, ok := m.Get("key25"); ok {
		t.Fatal("key25 still present after delete")
	}
	if v, ok := m.Get("key24"); !ok || v != 24 {
		t.Fatal("key24 damaged by neighbor delete")
	}
}

func TestResizeKeepsAllEntries(t *testing.T) {
	m := NewWithCapacity[int](2)
	const n = 1000
	for i := 0; i < n; i++ {
		m.Put(fmt.Sprintf("k%d", i), i)
	}
	if m.Len() != n {
		t.Fatalf("Len() = %d, want %d", m.Len(), n)
	}
	for i := 0; i < n; i++ {
		if v, ok := m.Get(fmt.Sprintf("k%d", i)); !ok || v != i {
			t.Fatalf("after resize Get(k%d) = %d,%v", i, v, ok)
		}
	}
	if lf := float64(m.Len()) / float64(m.Buckets()); lf > 0.75 {
		t.Errorf("load factor %.2f > 0.75: resize not happening", lf)
	}
}

func TestRange(t *testing.T) {
	m := New[int]()
	want := map[string]int{"a": 1, "b": 2, "c": 3}
	for k, v := range want {
		m.Put(k, v)
	}
	got := map[string]int{}
	m.Range(func(k string, v int) bool {
		got[k] = v
		return true
	})
	if len(got) != len(want) {
		t.Fatalf("Range visited %d entries, want %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("Range got[%q] = %d, want %d", k, got[k], v)
		}
	}
	// 提前终止
	count := 0
	m.Range(func(string, int) bool { count++; return false })
	if count != 1 {
		t.Errorf("Range with early-stop visited %d, want 1", count)
	}
}

func BenchmarkPut(b *testing.B) {
	m := New[int]()
	keys := make([]string, 1024)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Put(keys[i%1024], i)
	}
}

func BenchmarkGet(b *testing.B) {
	m := New[int]()
	keys := make([]string, 1024)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
		m.Put(keys[i], i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get(keys[i%1024])
	}
}
