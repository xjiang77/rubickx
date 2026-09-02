import java.util.HashMap;
import java.util.Map;

public class TwoSum {
    // 一次遍历 + HashMap：seen[值]=下标，命中 target-x 即返回。O(n)/O(n)。
    public static int[] twoSum(int[] nums, int target) {
        Map<Integer, Integer> seen = new HashMap<>();
        for (int i = 0; i < nums.length; i++) {
            int need = target - nums[i];
            if (seen.containsKey(need)) {
                return new int[] {seen.get(need), i};
            }
            seen.put(nums[i], i);
        }
        return new int[0];
    }
}
