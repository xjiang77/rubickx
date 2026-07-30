import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

// ============================================================
// 练习：Top K Frequent Elements (LC 347)
// 目标复杂度：时间 O(n)，空间 O(n)
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
        // TODO: 你来实现
        return new int[0];
    }
}
