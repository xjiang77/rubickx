import java.util.HashMap;
import java.util.Map;

// ============================================================
// 练习：Two Sum (LC 1)
// 目标复杂度：时间 O(n)，空间 O(n)
//
// 题目 · LC 1. Two Sum（两数之和）· Easy
// https://leetcode.cn/problems/two-sum/
//
//   给定一个整数数组 nums 和一个整数 target，返回和为 target 的两个数的下标。
//   可以假设每种输入只会对应一个答案，且同一个元素不能重复使用。
//   答案可以按任意顺序返回。
//
//   示例：
//     nums = [2,7,11,15], target = 9  ->  [0,1]   (nums[0] + nums[1] == 9)
//     nums = [3,2,4],     target = 6  ->  [1,2]
//     nums = [3,3],       target = 6  ->  [0,1]
//   约束：
//     - 2 <= nums.length <= 10^4
//     - -10^9 <= nums[i] <= 10^9
//     - -10^9 <= target <= 10^9
//     - 只存在一个有效答案
//   进阶：能否设计出时间复杂度优于 O(n^2) 的算法？
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 一次遍历，用哈希表记录「已见过的值 -> 下标」。
//   2. 每拿到一个 x，先查 target-x 是否已经在表里；命中就返回两个下标。
//   3. 先查再存，否则当 target = 2*x 时会拿自己去配自己。
//
// 跑测试： make practice-java
// 参考解在同目录的 TwoSum.java —— 卡死了再翻。
// ============================================================
public class TwoSumPractice {

    public static int[] twoSum(int[] nums, int target) {
        Map<Integer, Integer> seen = new HashMap<>();
        for (int i = 0; i < nums.length; i++) {
            int complement = target - nums[i];
            if (seen.containsKey(complement)) {
                return new int[] { seen.get(complement), i };
            }
            seen.put(nums[i], i);
        }
        return new int[0];
    }
}
