import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.PriorityQueue;
// ============================================================
// 练习：Top K Frequent Elements (LC 347)
// 目标复杂度：时间 O(n)，空间 O(n)
//
// 题目 · LC 347. Top K Frequent Elements（前 K 个高频元素）· Medium
// https://leetcode.cn/problems/top-k-frequent-elements/
//
//   给定一个整数数组 nums 和一个整数 k，返回其中出现频率前 k 高的元素。
//   可以按任意顺序返回答案。
//
//   示例：
//     nums = [1,1,1,2,2,3], k = 2   ->  [1,2]
//     nums = [1],           k = 1   ->  [1]
//   约束：
//     - 1 <= nums.length <= 10^5
//     - -10^4 <= nums[i] <= 10^4
//     - k 的取值范围是 [1, 数组中不相同的元素的个数]
//     - 题目数据保证答案唯一
//   进阶：所设计算法的时间复杂度必须优于 O(n log n)。
//
// 思路提示（想不起来再看，先自己憋一憋）：
//   1. 第一步永远是计数：map[值] -> 出现次数。
//   2. 堆的写法是 O(n log k)；桶排序能做到 O(n)。
//   3. 桶排序思路：出现次数最多不超过 n，所以开 n+1 个桶，buckets[c] 放所有出现 c 次的值。
//   4. 从高频桶往低频桶倒着扫，收满 k 个就停。
//   5. 结果顺序题目不做要求，所以测试里要先排序再比较。
//
// 跑测试： make practice-java
// 参考解在同目录的 TopKFrequent.java —— 卡死了再翻。
// ============================================================
public class TopKFrequentPractice {

    public static int[] topKFrequent(int[] nums, int k) {
        // 1. 计数
        Map<Integer, Integer> freqMap = new HashMap<>();
        for (int num : nums) {
            freqMap.put(num, freqMap.getOrDefault(num, 0) + 1);
        }
        // 2. 桶排序：下标=频次
        @SuppressWarnings("unchecked")
        List<Integer>[] buckets = new List[nums.length + 1];
        for (int i = 0; i < buckets.length; i++) {
            buckets[i] = new ArrayList<>();
        }
        for (Entry<Integer, Integer> entry : freqMap.entrySet()) {
            buckets[entry.getValue()].add(entry.getKey());
        }
        // 3. 从高频桶向低频收集 k 个。
        int[] result = new int[k];
        int idx = 0;
        for (int c = nums.length; c > 0 && idx < k; c--) {
            for (int num : buckets[c]) {
                result[idx++] = num;
                if (idx == k) {
                    return result;
                }
            }
        }
        return result;
    }
    //PriorityQueue<Entry<Integer, Integer>> minHeap = new PriorityQueue<>((a, b) -> a.getValue() - b.getValue());
    // for (Entry<Integer, Integer> entry : freqMap.entrySet()) {
    //     minHeap.offer(entry);
    //     if (minHeap.size() > k) {
    //         minHeap.poll();
    //     }
    // }
    // int[] result = new int[k];
    // for (int i = 0; i < k; i++) {
    //     result[i] = minHeap.poll().getKey();
    // }
}
