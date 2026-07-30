// ============================================================
// 练习：Valid Anagram (LC 242)
// 目标复杂度：时间 O(n)，空间 O(k)（k 为字符集大小）
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
