import java.util.HashSet;
import java.util.Set;

// ============================================================
// 练习：Longest Consecutive Sequence (LC 128)
// 目标复杂度：时间 O(n)，空间 O(n)
//
// 题目 · LC 128. Longest Consecutive Sequence（最长连续序列）· Medium
// https://leetcode.cn/problems/longest-consecutive-sequence/
//
//   给定一个未排序的整数数组 nums，找出数字连续的最长序列（不要求序列元素在原数组中连续）的长度。
//   要求设计并实现时间复杂度为 O(n) 的算法。
//
//   示例：
//     nums = [100,4,200,1,3,2]        ->  4   (最长连续序列是 1,2,3,4)
//     nums = [0,3,7,2,5,8,4,6,0,1]    ->  9
//     nums = []                       ->  0
//   约束：
//     - 0 <= nums.length <= 10^5
//     - -10^9 <= nums[i] <= 10^9
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 先把所有数塞进一个 set，查询 O(1)。
//   2. 关键剪枝：只在 x 是一段连续序列的「起点」时才展开，即 x-1 不在 set 里。
//   3. 从起点开始 x+1, x+2, ... 一直数到断掉，记录最长长度。
//   4. 有了这个剪枝，每个元素最多被展开访问一次，整体才是 O(n)；否则会退化成 O(n²)。
//   5. 空数组要返回 0。
//
// 跑测试： make practice-java
// 参考解在同目录的 LongestConsecutive.java —— 卡死了再翻。
// ============================================================
public class LongestConsecutivePractice {

    public static int longestConsecutive(int[] nums) {
        // TODO: 你来实现
        return 0;
    }
}
