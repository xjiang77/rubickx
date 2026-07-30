import java.util.HashSet;
import java.util.Set;

public class LongestConsecutive {
    // HashSet 去重,只从序列起点(x-1 不在集合)向右延伸,每个元素最多访问两次。O(n)/O(n)。
    public static int longestConsecutive(int[] nums) {
        Set<Integer> set = new HashSet<>();
        for (int x : nums) set.add(x);
        int best = 0;
        for (int x : set) {
            if (set.contains(x - 1)) continue; // 不是起点
            int len = 1;
            while (set.contains(x + len)) len++;
            best = Math.max(best, len);
        }
        return best;
    }
}
