import static org.junit.jupiter.api.Assertions.assertArrayEquals;

import org.junit.jupiter.api.Test;

public class ProductExceptSelfTest {
    @Test
    void basic() {
        assertArrayEquals(new int[] {24, 12, 8, 6}, ProductExceptSelf.productExceptSelf(new int[] {1, 2, 3, 4}));
    }

    @Test
    void withZero() {
        assertArrayEquals(new int[] {0, 0, 9, 0, 0}, ProductExceptSelf.productExceptSelf(new int[] {-1, 1, 0, -3, 3}));
    }

    @Test
    void twoElements() {
        assertArrayEquals(new int[] {3, 2}, ProductExceptSelf.productExceptSelf(new int[] {2, 3}));
    }
}
