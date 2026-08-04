import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

// ============================================================
// 练习：Group Anagrams (LC 49)
// 目标复杂度：时间 O(n·k·log k)（排序做 key）或 O(n·k)（计数做 key），空间 O(n·k)
//
// 题目 · LC 49. Group Anagrams（字母异位词分组）· Medium
// https://leetcode.cn/problems/group-anagrams/
//
//   给定一个字符串数组 strs，把互为字母异位词的字符串组合在一起。
//   可以按任意顺序返回结果列表，组内顺序也不做要求。
//   字母异位词：由重新排列源单词的所有字母得到的新单词。
//
//   示例：
//     strs = ["eat","tea","tan","ate","nat","bat"]
//       ->  [["bat"],["nat","tan"],["ate","eat","tea"]]
//     strs = [""]     ->  [[""]]
//     strs = ["a"]    ->  [["a"]]
//   约束：
//     - 1 <= strs.length <= 10^4
//     - 0 <= strs[i].length <= 100
//     - strs[i] 仅包含小写英文字母
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 核心是给每个字符串算一个「异位词之间相同」的规范 key。
//   2. 最直白的 key：把字符排序后的字符串。
//   3. 另一种 key：长度 26 的计数数组序列化成字符串，可以省掉排序的 log k。
//   4. 用 map[key] -> 列表 分组，最后把 map 的所有 value 收集成结果。
//   5. 结果的组顺序、组内顺序都不做要求，所以测试要先归一化再比较。
//
// 跑测试： make practice-java
// 参考解在同目录的 GroupAnagrams.java —— 卡死了再翻。
// ============================================================
public class GroupAnagramsPractice {

    public static List<List<String>> groupAnagrams(String[] strs) {
        // TODO: 你来实现
        return new ArrayList<>();
    }
}
