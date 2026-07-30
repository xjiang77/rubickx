import static org.junit.jupiter.api.Assertions.assertArrayEquals;

import java.util.Arrays;
import org.junit.jupiter.api.Test;

public class TopKFrequentTest {
    private static int[] sorted(int[] a) {
        int[] b = a.clone();
        Arrays.sort(b);
        return b;
    }

    @Test
    void basic() {
        assertArrayEquals(new int[] {1, 2}, sorted(TopKFrequent.topKFrequent(new int[] {1, 1, 1, 2, 2, 3}, 2)));
    }

    @Test
    void single() {
        assertArrayEquals(new int[] {1}, TopKFrequent.topKFrequent(new int[] {1}, 1));
    }

    @Test
    void sameFreq() {
        assertArrayEquals(new int[] {4, 5, 6}, sorted(TopKFrequent.topKFrequent(new int[] {4, 5, 6}, 3)));
    }
}
