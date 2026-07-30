import static org.junit.jupiter.api.Assertions.assertArrayEquals;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

// @Tag("practice") 让 `make test-java` 跳过这些还没写完的练习，
// `make practice-java` 则只跑它们。
@Tag("practice")
public class TwoSumPracticeTest {

    // 示例测试：先让这一个跑通，再照着往下补自己的场景。
    @Test
    void example() {
        assertArrayEquals(new int[] {0, 1}, TwoSumPractice.twoSum(new int[] {2, 7, 11, 15}, 9));
    }

    // TODO: 补充更多正常场景
    //   - 答案落在数组中间：[3,2,4] target=6
    //   - 两个值相同的数：[3,3] target=6
    //   - 长一点的数组，答案在末尾
    //
    // TODO: 想清楚这些 edge case 要不要测、期望是什么
    //   - 负数 / 0 参与求和
    //   - 无解时你的实现返回什么？空数组？null？测试应该把这个约定钉死
    //   - 返回的下标顺序有没有要求
}
