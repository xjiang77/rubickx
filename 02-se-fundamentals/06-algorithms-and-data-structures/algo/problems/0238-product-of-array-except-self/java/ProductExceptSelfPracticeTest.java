import static org.junit.jupiter.api.Assertions.assertArrayEquals;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

// @Tag("practice") 让 `make test-java` 跳过这些还没写完的练习，
// `make practice-java` 则只跑它们。
@Tag("practice")
public class ProductExceptSelfPracticeTest {

    // 示例测试：[1,2,3,4] -> [24,12,8,6]。
    @Test
    void example() {
        assertArrayEquals(new int[] {24, 12, 8, 6},
                ProductExceptSelfPractice.productExceptSelf(new int[] {1, 2, 3, 4}));
    }

    // TODO: 补充更多场景
    //   - 含一个 0：[-1,1,0,-3,3] -> [0,0,9,0,0]
    //   - 含两个 0：结果应该全是 0
    //   - 全负数、正负混合（注意符号）
    //
    // TODO: edge case
    //   - 长度 2 的数组：[a,b] -> [b,a]
    //   - 含 1 的数组
    //   - 溢出：Java 的 int 是 32 位，元素稍大就可能溢出 —— Go 那边 int 是 64 位，
    //     同样的输入两边结果会不一样，值得专门写个用例记录下来
}
