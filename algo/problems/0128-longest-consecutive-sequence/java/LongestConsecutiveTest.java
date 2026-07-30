import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;

public class LongestConsecutiveTest {
    @Test
    void basic() {
        assertEquals(4, LongestConsecutive.longestConsecutive(new int[] {100, 4, 200, 1, 3, 2}));
    }

    @Test
    void withDuplicates() {
        assertEquals(9, LongestConsecutive.longestConsecutive(new int[] {0, 3, 7, 2, 5, 8, 4, 6, 0, 1}));
    }

    @Test
    void empty() {
        assertEquals(0, LongestConsecutive.longestConsecutive(new int[] {}));
    }

    @Test
    void single() {
        assertEquals(1, LongestConsecutive.longestConsecutive(new int[] {5}));
    }
}
