import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

// @Tag("practice") 让 `make test-java` 跳过这些还没写完的练习，
// `make practice-java` 则只跑它们。
@Tag("practice")
public class ValidSudokuPracticeTest {

    private static final String[] VALID = {
        "53..7....",
        "6..195...",
        ".98....6.",
        "8...6...3",
        "4..8.3..1",
        "7...2...6",
        ".6....28.",
        "...419..5",
        "....8..79",
    };

    private static char[][] toBoard(String[] rows) {
        char[][] b = new char[9][];
        for (int i = 0; i < 9; i++) {
            b[i] = rows[i].toCharArray();
        }
        return b;
    }

    // 示例测试：一个合法数独应该返回 true。
    @Test
    void validBoard() {
        assertTrue(ValidSudokuPractice.isValidSudoku(toBoard(VALID)));
    }

    // TODO: 三类冲突各写一个用例（把上面的 VALID 复制一份、改一个字符就行）
    //   - 同一行出现重复数字
    //   - 同一列出现重复数字
    //   - 同一个 3x3 宫内重复，但行、列都不重复 <- 这条最容易漏
    //
    // TODO: edge case
    //   - 全 '.' 的空棋盘（应该是合法的）
    //   - 只填一个数字
    //   - 重复出现在棋盘最后一格
}
