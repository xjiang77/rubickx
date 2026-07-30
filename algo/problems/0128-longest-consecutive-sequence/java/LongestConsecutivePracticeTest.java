import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

// @Tag("practice") 让 `make test-java` 跳过这些还没写完的练习，
// `make practice-java` 则只跑它们。
@Tag("practice")
public class LongestConsecutivePracticeTest {

    // 示例测试：[100,4,200,1,3,2] 的最长连续段是 1,2,3,4，长度 4。
    @Test
    void example() {
        assertEquals(4, LongestConsecutivePractice.longestConsecutive(
                new int[] {100, 4, 200, 1, 3, 2}));
    }

    // TODO: 补充更多场景
    //   - 有重复元素：[1,2,0,1] -> 3（重复不该让长度变长）
    //   - 完全没有连续段：[10,30,50] -> 1
    //   - 整个数组就是一段连续：[5,4,3,2,1] -> 5
    //
    // TODO: edge case
    //   - 空数组 -> 0
    //   - 单元素 -> 1
    //   - 负数跨 0：[-2,-1,0,1] -> 4
    //   - 两段等长的连续序列
    //
    // TODO（可选）：造一个 10 万级别、最坏形状的输入，验证你的实现真的是 O(n)
    //   —— 如果漏了「只从起点展开」的剪枝，这里会明显变慢
}
