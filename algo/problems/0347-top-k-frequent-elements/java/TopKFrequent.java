import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class TopKFrequent {
    // 计数 + 桶排序:merge 计数,下标=频次的桶,从高频向低频收集 k 个。O(n)/O(n)。
    public static int[] topKFrequent(int[] nums, int k) {
        Map<Integer, Integer> counts = new HashMap<>();
        for (int x : nums) counts.merge(x, 1, Integer::sum);
        List<List<Integer>> buckets = new ArrayList<>(nums.length + 1);
        for (int i = 0; i <= nums.length; i++) buckets.add(new ArrayList<>());
        for (Map.Entry<Integer, Integer> e : counts.entrySet()) {
            buckets.get(e.getValue()).add(e.getKey());
        }
        int[] res = new int[k];
        int idx = 0;
        for (int c = nums.length; c > 0 && idx < k; c--) {
            for (int x : buckets.get(c)) {
                res[idx++] = x;
                if (idx == k) return res;
            }
        }
        return res;
    }
}
