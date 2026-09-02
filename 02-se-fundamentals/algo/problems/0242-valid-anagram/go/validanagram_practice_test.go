//go:build practice

package validanagram

import "testing"

// 示例测试。
func TestIsAnagramPractice(t *testing.T) {
	s, tt := "anagram", "nagaram"
	if got := IsAnagramPractice(s, tt); !got {
		t.Errorf("IsAnagramPractice(%q, %q) = false, want true", s, tt)
	}

	// TODO: 补充更多场景
	//   - 字符集相同但个数不同："aacc" vs "ccac" -> false
	//   - 完全不同的字符："rat" vs "car" -> false
	//   - 长度不同："a" vs "ab" -> false
	//
	// TODO: edge case
	//   - 两个空串 -> true
	//   - 单字符相同 / 不同
	//   - 自己和自己："abc" vs "abc" -> true
	//   - 多字节字符："你好" vs "好你" —— 你的实现按 rune 还是按 byte 处理？
	//     （Go 里 len("你好") 是 6 不是 2，这里最容易翻车）
}
