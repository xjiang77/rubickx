import static org.junit.jupiter.api.Assertions.assertEquals;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import org.junit.jupiter.api.Test;

public class GroupAnagramsTest {
    private static List<List<String>> norm(List<List<String>> groups) {
        List<List<String>> out = new ArrayList<>();
        for (List<String> g : groups) {
            List<String> cp = new ArrayList<>(g);
            Collections.sort(cp);
            out.add(cp);
        }
        out.sort((a, b) -> String.join(",", a).compareTo(String.join(",", b)));
        return out;
    }

    @Test
    void basic() {
        List<List<String>> res =
                GroupAnagrams.groupAnagrams(new String[] {"eat", "tea", "tan", "ate", "nat", "bat"});
        List<List<String>> want =
                List.of(List.of("bat"), List.of("nat", "tan"), List.of("ate", "eat", "tea"));
        assertEquals(norm(want), norm(res));
    }
}
