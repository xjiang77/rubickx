// ============================================================
// 练习：Product of Array Except Self (LC 238)
// 目标复杂度：时间 O(n)，除返回值外空间 O(1)
//
// 题目 · LC 238. Product of Array Except Self（除自身以外数组的乘积）· Medium
// https://leetcode.cn/problems/product-of-array-except-self/
//
//   给定一个整数数组 nums，返回数组 answer，其中 answer[i] 等于 nums 中除 nums[i] 之外其余各元素的乘积。
//   题目数据保证任何前缀或后缀的乘积都在 32 位整数范围内。
//   请不要使用除法，且在 O(n) 时间内完成。
//
//   示例：
//     nums = [1,2,3,4]        ->  [24,12,8,6]
//     nums = [-1,1,0,-3,3]    ->  [0,0,9,0,0]
//   约束：
//     - 2 <= nums.length <= 10^5
//     - -30 <= nums[i] <= 30
//     - 保证任何前缀或后缀的乘积都在 32 位整数范围内
//   进阶：能否只用 O(1) 的额外空间完成？（返回的答案数组不计入额外空间）
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 题目禁止用除法（而且有 0 时除法本来就会炸）。
//   2. 答案 = 左边所有数的积 × 右边所有数的积。
//   3. 第一遍从左往右，把「前缀积」直接写进结果数组。
//   4. 第二遍从右往左，用一个滚动变量累乘「后缀积」，乘到结果数组上，这样不额外开数组。
//   5. 注意两遍都是「不含自己」：更新滚动变量的时机要在写结果之后。
//
// 跑测试： make practice-java
// 参考解在同目录的 ProductExceptSelf.java —— 卡死了再翻。
// ============================================================
public class ProductExceptSelfPractice {

    public static int[] productExceptSelf(int[] nums) {
        // TODO: 你来实现
        return new int[0];
    }
}
