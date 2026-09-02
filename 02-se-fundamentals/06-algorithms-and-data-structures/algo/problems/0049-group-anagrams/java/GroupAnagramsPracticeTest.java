import static org.junit.jupiter.api.Assertions.assertEquals;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

// @Tag("practice") 让 `make test-java` 跳过这些还没写完的练习，
// `make practice-java` 则只跑它们。
@Tag("practice")
public class GroupAnagramsPracticeTest {

    // 分组顺序和组内顺序都不做要求，比较前先归一化。
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

    // 示例测试。
    @Test
    void example() {
        List<List<String>> got =
                GroupAnagramsPractice.groupAnagrams(
                        new String[] {"eat", "tea", "tan", "ate", "nat", "bat"});
        List<List<String>> want =
                List.of(List.of("bat"), List.of("nat", "tan"), List.of("ate", "eat", "tea"));
        System.out.println("got: " + got);
        System.out.println("want: " + want);
        assertEquals(norm(want), norm(got));
    }

    // TODO: 补充更多场景
    //   - 每个词自成一组（没有任何异位词）
    //   - 所有词属于同一组
    //   - 有重复的输入词，比如 ["ab","ab"]，它们应该在同一组且都保留
    //
    // TODO: edge case
    //   - 空输入 new String[] {}
    //   - 含空字符串 ""
    //   - 单字符 ["a"]
    //   - 长度相同但字符不同的词，会不会被你的 key 误判成一组？
}
