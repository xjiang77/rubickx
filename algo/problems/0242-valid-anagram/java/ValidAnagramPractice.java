// ============================================================
// 练习：Valid Anagram (LC 242)
// 目标复杂度：时间 O(n)，空间 O(k)（k 为字符集大小）
//
// 题目 · LC 242. Valid Anagram（有效的字母异位词）· Easy
// https://leetcode.cn/problems/valid-anagram/
//
//   给定两个字符串 s 和 t，判断 t 是否是 s 的字母异位词。
//   字母异位词：t 由 s 的所有字母重新排列组成（每个字符出现次数完全相同）。
//
//   示例：
//     s = "anagram", t = "nagaram"  ->  true
//     s = "rat",     t = "car"      ->  false
//   约束：
//     - 1 <= s.length, t.length <= 5 * 10^4
//     - s 和 t 仅包含小写字母
//   进阶：如果输入字符串包含 Unicode 字符怎么办？你的解法能否适配这种情况？
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 先比长度，不等直接 false —— 顺手挡掉一大类输入。
//   2. 统计 s 的字符频次，再用 t 去抵消；任一字符计数变负就 false。
//   3. 假定只有小写字母时，长度 26 的数组比 HashMap 快很多。
//   4. 要支持 Unicode 就得用 map，并且注意 Go 的 len(s) 是字节数不是字符数，得按 rune 遍历。
//
// 跑测试： make practice-java
// 参考解在同目录的 ValidAnagram.java —— 卡死了再翻。
// ============================================================
public class ValidAnagramPractice {

    public static boolean isAnagram(String s, String t) {
        // TODO: 你来实现
        return false;
    }
}
