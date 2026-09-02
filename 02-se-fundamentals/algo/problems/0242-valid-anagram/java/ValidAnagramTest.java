import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;

public class ValidAnagramTest {
    @Test
    void valid() { assertTrue(ValidAnagram.isAnagram("anagram", "nagaram")); }

    @Test
    void invalid() { assertFalse(ValidAnagram.isAnagram("rat", "car")); }

    @Test
    void diffLen() { assertFalse(ValidAnagram.isAnagram("a", "ab")); }
}
