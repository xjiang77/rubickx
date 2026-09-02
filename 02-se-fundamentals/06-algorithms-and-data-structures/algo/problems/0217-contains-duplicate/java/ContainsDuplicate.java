import java.util.HashSet;
import java.util.Set;

public class ContainsDuplicate {
    // HashSet.add 返回 false 即表示已存在 -> 命中重复。O(n)/O(n)。
    public static boolean containsDuplicate(int[] nums) {
        Set<Integer> seen = new HashSet<>();
        for (int x : nums) {
            if (!seen.add(x)) {
                return true;
            }
        }
        return false;
    }
}
