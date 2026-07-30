import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

// @Tag("practice") 让 `make test-java` 跳过这些还没写完的练习，
// `make practice-java` 则只跑它们。
@Tag("practice")
public class ContainsDuplicatePracticeTest {

    // 示例测试。
    @Test
    void example() {
        assertTrue(ContainsDuplicatePractice.containsDuplicate(new int[] {1, 2, 3, 1}));
    }

    // TODO: 补充更多场景
    //   - 全不重复：[1,2,3,4] -> false
    //   - 相邻重复：[1,1] -> true
    //   - 重复出现在数组最后两个位置（确认你没提前 return 漏掉）
    //
    // TODO: edge case
    //   - 空数组 -> false
    //   - 单元素 -> false
    //   - 含负数、含 0
    //   - 全是同一个值的长数组
}
