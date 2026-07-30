import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

// @Tag("practice") 让 `make test-java` 跳过这些还没写完的练习，
// `make practice-java` 则只跑它们。
@Tag("practice")
public class ValidAnagramPracticeTest {

    // 示例测试。
    @Test
    void example() {
        assertTrue(ValidAnagramPractice.isAnagram("anagram", "nagaram"));
    }

    // TODO: 补充更多场景
    //   - 字符集相同但个数不同："aacc" vs "ccac" -> false
    //   - 完全不同的字符："rat" vs "car" -> false
    //   - 长度不同："a" vs "ab" -> false
    //
    // TODO: edge case
    //   - 两个空串 -> true
    //   - 单字符相同 / 不同
    //   - 自己和自己："abc" vs "abc" -> true
    //   - 非小写字母（大写、数字、中文）：如果你用了 count[c - 'a']，
    //     这类输入会直接数组越界 —— 先想清楚约定，再决定要不要测
}
