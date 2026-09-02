import static org.junit.jupiter.api.Assertions.assertArrayEquals;

import java.util.Arrays;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

// @Tag("practice") 让 `make test-java` 跳过这些还没写完的练习，
// `make practice-java` 则只跑它们。
@Tag("practice")
public class TopKFrequentPracticeTest {

    // 返回顺序题目不做要求，比较前先排序。
    // 示例测试：[1,1,1,2,2,3], k=2 -> {1,2}。
    @Test
    void example() {
        int[] got = TopKFrequentPractice.topKFrequent(new int[] {1, 1, 1, 2, 2, 3}, 2);
        Arrays.sort(got);
        assertArrayEquals(new int[] {1, 2}, got);
    }

    @Test
    void testK1() {
        int[] got = TopKFrequentPractice.topKFrequent(new int[] {1}, 1);
        assertArrayEquals(new int[] {1}, got);
    }


    @Test
    void testSingleElementArray() {
        int[] got = TopKFrequentPractice.topKFrequent(new int[] {1}, 1);
        assertArrayEquals(new int[] {1}, got);
    }

    @Test
    void testBoundaryCase() {
        int[] got = TopKFrequentPractice.topKFrequent(new int[] {1, 1, 2, 2, 3}, 2);
        assertArrayEquals(new int[] {1, 2}, got);
    }

    @Test
    void testMultipleElementsWithSameFrequency() {
        int[] got = TopKFrequentPractice.topKFrequent(new int[] {1, 1, 2, 2, 3}, 2);
        assertArrayEquals(new int[] {1, 2}, got);
    }
    // TODO: 补充更多场景
    //   - k=1，只取最高频
    //   - k 等于不同元素的个数（等于全取）
    //   - 所有元素频次都一样，比如 [1,2,3] k=2（答案不唯一，测试怎么写才稳？）
    //
    // TODO: edge case
    //   - 单元素数组 [1], k=1
    //   - 含负数
    //   - 频次并列卡在第 k 名边界上，比如 [1,1,2,2,3] k=2
    //   - 检查返回数组长度必须恰好是 k（桶排序写法很容易多返回几个）
}
