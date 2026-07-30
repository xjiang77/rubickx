import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;

public class ValidSudokuTest {
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
        char[][] board = new char[9][];
        for (int i = 0; i < 9; i++) board[i] = rows[i].toCharArray();
        return board;
    }

    @Test
    void validBoard() {
        assertTrue(ValidSudoku.isValidSudoku(toBoard(VALID)));
    }

    @Test
    void boxDuplicate() {
        String[] rows = VALID.clone();
        rows[0] = "8" + rows[0].substring(1); // (0,0) 5→8,与 (2,2) 的 8 同 box
        assertFalse(ValidSudoku.isValidSudoku(toBoard(rows)));
    }

    @Test
    void rowDuplicate() {
        String[] rows = VALID.clone();
        rows[0] = rows[0].substring(0, 2) + "5" + rows[0].substring(3); // 第 0 行两个 5
        assertFalse(ValidSudoku.isValidSudoku(toBoard(rows)));
    }
}
