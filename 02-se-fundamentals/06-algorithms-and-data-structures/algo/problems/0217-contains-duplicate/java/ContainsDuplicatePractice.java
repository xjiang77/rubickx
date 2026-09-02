import java.util.HashSet;
import java.util.Set;

// ============================================================
// 练习：Contains Duplicate (LC 217)
// 目标复杂度：时间 O(n)，空间 O(n)
//
// 题目 · LC 217. Contains Duplicate（存在重复元素）· Easy
// https://leetcode.cn/problems/contains-duplicate/
//
//   给定一个整数数组 nums，如果任一值在数组中出现至少两次，返回 true；
//   如果数组中每个元素互不相同，返回 false。
//
//   示例：
//     nums = [1,2,3,1]          ->  true
//     nums = [1,2,3,4]          ->  false
//     nums = [1,1,1,3,3,4,3,2,4,2]  ->  true
//   约束：
//     - 1 <= nums.length <= 10^5
//     - -10^9 <= nums[i] <= 10^9
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 边遍历边往 set 里放，放之前先看在不在。
//   2. Java 里 HashSet.add 的返回值本身就是判重信号：返回 false 说明已存在。
//   3. Go 里习惯用 map[int]struct{} 当 set，struct{} 不占空间。
//   4. 另一条路：排序后比较相邻元素，时间 O(n log n) 但空间 O(1) —— 可以两种都写写对比。
//
// 跑测试： make practice-java
// 参考解在同目录的 ContainsDuplicate.java —— 卡死了再翻。
// ============================================================
public class ContainsDuplicatePractice {

    public static boolean containsDuplicate(int[] nums) {
        Set<Integer> seen = new HashSet<>();
        for (int n : nums) {
            if (seen.contains(n)) {
                return true;
            }
            seen.add(n);
        }
        return false;
    }
}
