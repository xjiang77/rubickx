import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;

public class ContainsDuplicateTest {
    @Test
    void hasDup() { assertTrue(ContainsDuplicate.containsDuplicate(new int[] {1, 2, 3, 1})); }

    @Test
    void unique() { assertFalse(ContainsDuplicate.containsDuplicate(new int[] {1, 2, 3, 4})); }

    @Test
    void empty() { assertFalse(ContainsDuplicate.containsDuplicate(new int[] {})); }
}
