package dynamicarray

import "testing"

// 契约:
//   New[T]() 创建空数组;Len()==0, Cap()>=0
//   Append 均摊 O(1),容量不足时按 2x 扩容(初始扩容到至少 1)
//   Get/Set 越界必须 panic
//   Insert(i, v) 允许 i==Len()(等价 Append);RemoveAt 返回被删元素
//   底层不允许用 built-in append 偷懒 —— 手动管理 backing slice 的 len/cap 语义

func TestEmptyArray(t *testing.T) {
	a := New[int]()
	if a.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", a.Len())
	}
}

func TestAppendAndGet(t *testing.T) {
	a := New[string]()
	words := []string{"go", "is", "expressive", "concise", "clean"}
	for _, w := range words {
		a.Append(w)
	}
	if a.Len() != len(words) {
		t.Fatalf("Len() = %d, want %d", a.Len(), len(words))
	}
	for i, w := range words {
		if got := a.Get(i); got != w {
			t.Errorf("Get(%d) = %q, want %q", i, got, w)
		}
	}
}

func TestGrowthDoubling(t *testing.T) {
	a := New[int]()
	prevCap := a.Cap()
	grows := 0
	for i := 0; i < 1000; i++ {
		a.Append(i)
		if c := a.Cap(); c != prevCap {
			if prevCap > 0 && c < prevCap*2 {
				t.Fatalf("grow from cap %d to %d: expected at least doubling", prevCap, c)
			}
			prevCap = c
			grows++
		}
	}
	if grows > 12 {
		t.Errorf("1000 appends caused %d grows; doubling should need ~10", grows)
	}
	if a.Cap() < a.Len() {
		t.Fatalf("Cap() %d < Len() %d", a.Cap(), a.Len())
	}
}

func TestSet(t *testing.T) {
	a := New[int]()
	a.Append(1)
	a.Append(2)
	a.Set(1, 20)
	if got := a.Get(1); got != 20 {
		t.Errorf("after Set(1,20), Get(1) = %d", got)
	}
}

func TestInsert(t *testing.T) {
	a := New[int]()
	for _, v := range []int{1, 2, 4} {
		a.Append(v)
	}
	a.Insert(2, 3)            // 中间插入
	a.Insert(0, 0)            // 头部插入
	a.Insert(a.Len(), 5)      // 尾部插入 == Append
	want := []int{0, 1, 2, 3, 4, 5}
	if a.Len() != len(want) {
		t.Fatalf("Len() = %d, want %d", a.Len(), len(want))
	}
	for i, w := range want {
		if got := a.Get(i); got != w {
			t.Errorf("Get(%d) = %d, want %d", i, got, w)
		}
	}
}

func TestRemoveAt(t *testing.T) {
	a := New[int]()
	for _, v := range []int{10, 20, 30, 40} {
		a.Append(v)
	}
	if got := a.RemoveAt(1); got != 20 {
		t.Errorf("RemoveAt(1) = %d, want 20", got)
	}
	if got := a.RemoveAt(0); got != 10 {
		t.Errorf("RemoveAt(0) = %d, want 10", got)
	}
	want := []int{30, 40}
	if a.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", a.Len())
	}
	for i, w := range want {
		if got := a.Get(i); got != w {
			t.Errorf("Get(%d) = %d, want %d", i, got, w)
		}
	}
}

func TestPanicsOnOutOfRange(t *testing.T) {
	cases := []struct {
		name string
		f    func(a *Array[int])
	}{
		{"Get(-1)", func(a *Array[int]) { a.Get(-1) }},
		{"Get(Len())", func(a *Array[int]) { a.Get(a.Len()) }},
		{"Set(Len())", func(a *Array[int]) { a.Set(a.Len(), 0) }},
		{"Insert(Len()+1)", func(a *Array[int]) { a.Insert(a.Len()+1, 0) }},
		{"RemoveAt(-1)", func(a *Array[int]) { a.RemoveAt(-1) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := New[int]()
			a.Append(1)
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected panic", tc.name)
				}
			}()
			tc.f(a)
		})
	}
}

func BenchmarkAppend(b *testing.B) {
	a := New[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Append(i)
	}
}
