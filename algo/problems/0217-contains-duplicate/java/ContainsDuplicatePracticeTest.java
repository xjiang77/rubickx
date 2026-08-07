import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.assertFalse;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

// @Tag("practice") 让 `make test-java` 跳过这些还没写完的练习，
// `make practice-java` 则只跑它们。
@Tag("practice")
public class ContainsDuplicatePracticeTest {

    // 示例测试。
    @Test
    void testExample() {
        assertTrue(ContainsDuplicatePractice.containsDuplicate(new int[] {1, 2, 3, 1}));
    }

    @Test
    void testAllDifferent() {
        assertFalse(ContainsDuplicatePractice.containsDuplicate(new int[] {1, 2, 3, 4}));
    }

    @Test
    void testAdjacentDuplicate() {
        assertTrue(ContainsDuplicatePractice.containsDuplicate(new int[] {1, 1, 2, 3}));
    }

    @Test
    void testDuplicateAtEnd() {
        assertTrue(ContainsDuplicatePractice.containsDuplicate(new int[] {1, 2, 3, 1}));
    }

    @Test
    void testEmptyArray() {
        assertFalse(ContainsDuplicatePractice.containsDuplicate(new int[] {}));
    }

    @Test
    void testSingleElement() {
        assertFalse(ContainsDuplicatePractice.containsDuplicate(new int[] {1}));
    }

    @Test
    void testNegativeNumbers() {
        assertFalse(ContainsDuplicatePractice.containsDuplicate(new int[] {-1, -2, -3, -4}));
    }

    @Test
    void testZero() {
        assertTrue(ContainsDuplicatePractice.containsDuplicate(new int[] {0, 0, 0, 0}));
    }

    @Test
    void testAllSameValue() {
        assertTrue(ContainsDuplicatePractice.containsDuplicate(new int[] {1, 1, 1, 1}));
    }

    @Test
    void testLargeArray() {
        assertFalse(ContainsDuplicatePractice.containsDuplicate(new int[] {1, 2, 3, 4, 5, 6, 7, 8, 9, 10}));
    }

    @Test
    void testLargeArrayWithDuplicate() {
        assertTrue(ContainsDuplicatePractice.containsDuplicate(new int[] {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1}));
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
