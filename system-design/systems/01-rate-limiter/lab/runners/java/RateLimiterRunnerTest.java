import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStreamWriter;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;

public class RateLimiterRunnerTest {
    private static final Path FIXTURES = Path.of(
            "systems/01-rate-limiter/lab/fixtures/core-parity.json");

    private static List<Map<String, Object>> runLines(String... lines)
            throws IOException, InterruptedException {
        Process process = new ProcessBuilder("java", "-cp", ".build/java", "RateLimiterRunner").start();
        try (BufferedWriter stdin = new BufferedWriter(
                new OutputStreamWriter(process.getOutputStream(), StandardCharsets.UTF_8))) {
            for (String line : lines) {
                stdin.write(line);
                stdin.newLine();
            }
        }
        List<Map<String, Object>> output = new ArrayList<>();
        try (BufferedReader stdout = new BufferedReader(
                new InputStreamReader(process.getInputStream(), StandardCharsets.UTF_8))) {
            String line;
            while ((line = stdout.readLine()) != null) {
                output.add(object(TestJson.parse(line)));
            }
        }
        String error = new String(process.getErrorStream().readAllBytes(), StandardCharsets.UTF_8);
        assertEquals(0, process.waitFor(), error);
        return output;
    }

    private static String request(String algorithm, Map<String, Object> config, List<Object> timeline) {
        return TestJson.stringify(map(
                "scenarioId", "contract-test",
                "algorithm", algorithm,
                "language", "java",
                "config", config,
                "requestTimeline", timeline,
                "storeMode", "memory"));
    }

    private static List<Object> burstTimeline() {
        return List.of(
                map("atMs", 0, "cost", 1, "key", "alice"),
                map("atMs", 0, "cost", 1, "key", "alice"),
                map("atMs", 0, "cost", 1, "key", "alice"),
                map("atMs", 1000, "cost", 1, "key", "alice"));
    }

    @Test
    void jsonlSeamRunsTokenBucketAndRecoversAfterMalformedJson() throws Exception {
        List<Map<String, Object>> output = runLines(
                "not json",
                request(
                        "token-bucket",
                        map("capacity", 2, "ratePerSecond", 1),
                        burstTimeline()));
        assertEquals(2, output.size());
        assertEquals("invalid_json", errorCode(output.get(0)));
        assertEquals(List.of(true, true, false, true), allowed(output.get(1)));
        List<Object> events = array(output.get(1).get("events"));
        Map<String, Object> last = object(events.get(events.size() - 1));
        assertEquals("token.decision", last.get("stepId"));
        assertEquals(
                List.of("seq", "stepId", "actor", "timestampMs", "before", "after", "decision", "reason"),
                new ArrayList<>(last.keySet()));
    }

    @Test
    void jsonlSeamSupportsAllWindowAndLeakyAlgorithms() throws Exception {
        List<Map<String, Object>> output = runLines(
                request("fixed-window", map("limit", 2, "windowMs", 1000), burstTimeline()),
                request("sliding-window-log", map("limit", 2, "windowMs", 1000), burstTimeline()),
                request("sliding-window-counter", map("limit", 2, "windowMs", 1000), burstTimeline()),
                request("leaky-bucket", map("capacity", 2, "ratePerSecond", 1), burstTimeline()));
        assertEquals(List.of(true, true, false, true), allowed(output.get(0)));
        assertEquals(List.of(true, true, false, true), allowed(output.get(1)));
        assertEquals(
                List.of(true, true, false, false),
                allowed(output.get(2)),
                "the previous window still has weight 1 at the exact boundary");
        assertEquals(List.of(true, true, false, true), allowed(output.get(3)));
    }

    @Test
    void invalidRequestDoesNotStopFollowingLine() throws Exception {
        List<Map<String, Object>> output = runLines(
                request("fixed-window", map("limit", 0, "windowMs", 1000), List.of()),
                request(
                        "fixed-window",
                        map("limit", 1, "windowMs", 1000),
                        List.of(map("atMs", 0, "cost", 1, "key", "alice"))));
        assertEquals("invalid_request", errorCode(output.get(0)));
        assertEquals(List.of(true), allowed(output.get(1)));
    }

    @Test
    void sharedFixturesCoverEveryAlgorithmBoundaryFractionalCostAndTimeJump() throws Exception {
        List<Object> fixtures = array(TestJson.parse(Files.readString(FIXTURES)));
        for (Object rawFixture : fixtures) {
            Map<String, Object> fixture = object(rawFixture);
            Map<String, Object> response = runLines(request(
                    (String) fixture.get("algorithm"),
                    object(fixture.get("config")),
                    array(fixture.get("requestTimeline")))).get(0);
            assertEquals(
                    fixture.get("expectedAllowed"),
                    allowed(response),
                    fixture.get("name") + " allowed decisions");
            assertNumbersEqual(
                    array(fixture.get("expectedRemaining")),
                    remaining(response),
                    (String) fixture.get("name"));
            List<Object> decisions = array(response.get("decisions"));
            Map<String, Object> last = object(decisions.get(decisions.size() - 1));
            assertEquals(
                    fixture.get("expectedLastReason"),
                    last.get("reason"),
                    (String) fixture.get("name"));
            assertEquals(
                    ((Number) fixture.get("expectedLastRetryAfterMs")).doubleValue(),
                    ((Number) last.get("retryAfterMs")).doubleValue(),
                    0.000001,
                    fixture.get("name") + " retryAfterMs");
            if (fixture.containsKey("expectedLastResetAtMs")) {
                assertEquals(
                        ((Number) fixture.get("expectedLastResetAtMs")).doubleValue(),
                        ((Number) last.get("resetAtMs")).doubleValue(),
                        0.000001,
                        fixture.get("name") + " resetAtMs");
            }
        }
    }

    @Test
    void emptyTimelineIsASuccessfulNoopForEveryAlgorithm() throws Exception {
        List<Map<String, Object>> output = runLines(
                request("fixed-window", map("limit", 2, "windowMs", 1000), List.of()),
                request("sliding-window-log", map("limit", 2, "windowMs", 1000), List.of()),
                request("sliding-window-counter", map("limit", 2, "windowMs", 1000), List.of()),
                request("token-bucket", map("capacity", 2, "ratePerSecond", 1), List.of()),
                request("leaky-bucket", map("capacity", 2, "ratePerSecond", 1), List.of()));
        for (Map<String, Object> response : output) {
            assertTrue(array(response.get("events")).isEmpty());
            assertTrue(array(response.get("decisions")).isEmpty());
        }
    }

    @Test
    void negativeAndNonMonotonicTimeAreRejected() throws Exception {
        List<Map<String, Object>> output = runLines(
                request(
                        "fixed-window",
                        map("limit", 2, "windowMs", 1000),
                        List.of(map("atMs", -1, "cost", 1, "key", "alice"))),
                request(
                        "fixed-window",
                        map("limit", 2, "windowMs", 1000),
                        List.of(
                                map("atMs", 2, "cost", 1, "key", "alice"),
                                map("atMs", 1, "cost", 1, "key", "alice"))));
        assertEquals("invalid_request", errorCode(output.get(0)));
        assertEquals("invalid_request", errorCode(output.get(1)));
    }

    @Test
    void fractionalTimeAndMoreThanOneHundredItemsAreRejected() throws Exception {
        List<Object> oversized = new ArrayList<>();
        for (int index = 0; index < 101; index++) {
            oversized.add(map("atMs", index, "cost", 1, "key", "alice"));
        }
        List<Map<String, Object>> output = runLines(
                request(
                        "fixed-window",
                        map("limit", 2, "windowMs", 1000),
                        List.of(map("atMs", 0.5, "cost", 1, "key", "alice"))),
                request("fixed-window", map("limit", 2, "windowMs", 1000), oversized));
        assertEquals("invalid_request", errorCode(output.get(0)));
        assertEquals("invalid_request", errorCode(output.get(1)));
    }

    @Test
    void timeAboveJavaScriptMaxSafeIntegerIsRejected() throws Exception {
        Map<String, Object> response = runLines(request(
                "fixed-window",
                map("limit", 2, "windowMs", 1000),
                List.of(map(
                        "atMs", 9_007_199_254_740_992L,
                        "cost", 1,
                        "key", "alice")))).get(0);
        assertEquals("invalid_request", errorCode(response));
        assertTrue(
                ((String) object(response.get("error")).get("message"))
                        .contains("safe integer milliseconds"));
    }

    @Test
    void eachRequestTraceIsStateContinuous() throws Exception {
        List<Object> timeline = List.of(
                map("atMs", 0, "cost", 1, "key", "alice"),
                map("atMs", 1000, "cost", 1, "key", "alice"));
        List<Map<String, Object>> responses = runLines(
                request("fixed-window", map("limit", 2, "windowMs", 1000), timeline),
                request("token-bucket", map("capacity", 2, "ratePerSecond", 1), timeline),
                request("leaky-bucket", map("capacity", 2, "ratePerSecond", 1), timeline));
        for (Map<String, Object> response : responses) {
            List<Object> events = array(response.get("events"));
            assertEquals(4, events.size());
            for (int index = 0; index < events.size(); index += 2) {
                assertEquals(
                        object(events.get(index)).get("after"),
                        object(events.get(index + 1)).get("before"));
            }
        }
    }

    @Test
    void nonPositiveOrFractionalWindowAndNonPositiveBucketConfigAreRejected() throws Exception {
        List<Map<String, Object>> output = runLines(
                request("fixed-window", map("limit", 0, "windowMs", 1000), List.of()),
                request("fixed-window", map("limit", 2, "windowMs", 0.5), List.of()),
                request("token-bucket", map("capacity", 2, "ratePerSecond", 0), List.of()));
        assertEquals("invalid_request", errorCode(output.get(0)));
        assertEquals("invalid_request", errorCode(output.get(1)));
        assertEquals("invalid_request", errorCode(output.get(2)));
    }

    @Test
    void windowAboveJavaScriptMaxSafeIntegerIsRejected() throws Exception {
        Map<String, Object> response = runLines(request(
                "fixed-window",
                map("limit", 2, "windowMs", 9_007_199_254_740_992L),
                List.of())).get(0);
        assertEquals("invalid_request", errorCode(response));
        assertTrue(
                ((String) object(response.get("error")).get("message"))
                        .contains("positive safe integer"));
    }

    @Test
    void unsafeQuantitiesAreRejectedAtThePublicSeam() throws Exception {
        List<Object> oneCost = List.of(map("atMs", 0, "cost", 1, "key", "alice"));
        List<Map<String, Object>> output = runLines(
                request("fixed-window", map("limit", 1e308, "windowMs", 1000), oneCost),
                request("token-bucket", map("capacity", 1e308, "ratePerSecond", 1), oneCost),
                request("token-bucket", map("capacity", 1, "ratePerSecond", 1e308), oneCost),
                request(
                        "fixed-window",
                        map("limit", 2, "windowMs", 1000),
                        List.of(map("atMs", 0, "cost", 1e308, "key", "alice"))),
                request("token-bucket", map("capacity", 1, "ratePerSecond", 5e-324), oneCost));
        for (Map<String, Object> response : output) {
            assertEquals("invalid_request", errorCode(response));
        }
    }

    @Test
    void maxSafeQuantitiesAreAcceptedAtThePublicSeam() throws Exception {
        long maximum = 9_007_199_254_740_991L;
        List<Map<String, Object>> output = runLines(
                request(
                        "fixed-window",
                        map("limit", maximum, "windowMs", 1000),
                        List.of(map("atMs", 0, "cost", maximum, "key", "alice"))),
                request(
                        "token-bucket",
                        map("capacity", maximum, "ratePerSecond", maximum),
                        List.of(map("atMs", 0, "cost", maximum, "key", "alice"))));
        assertEquals(List.of(true), allowed(output.get(0)));
        assertEquals(List.of(true), allowed(output.get(1)));
    }

    @Test
    void keysAboveOneHundredTwentyEightUtf8BytesAreRejected() throws Exception {
        List<Map<String, Object>> output = runLines(
                request(
                        "fixed-window",
                        map("limit", 1, "windowMs", 1000),
                        List.of(map("atMs", 0, "cost", 1, "key", "a".repeat(129)))),
                request(
                        "fixed-window",
                        map("limit", 1, "windowMs", 1000),
                        List.of(map("atMs", 0, "cost", 1, "key", "界".repeat(43)))));
        assertEquals("invalid_request", errorCode(output.get(0)));
        assertEquals("invalid_request", errorCode(output.get(1)));
    }

    @Test
    void multibyteKeyAtOneHundredTwentyEightUtf8BytesIsAccepted() throws Exception {
        String key = "界".repeat(42) + "ab";
        Map<String, Object> response = runLines(request(
                "fixed-window",
                map("limit", 1, "windowMs", 1000),
                List.of(map("atMs", 0, "cost", 1, "key", key)))).get(0);
        assertEquals(List.of(true), allowed(response));
        assertEquals(key, object(array(response.get("events")).get(0)).get("actor"));
    }

    @Test
    void costAndKeyAreRequiredAtThePublicSeam() throws Exception {
        List<Map<String, Object>> output = runLines(
                request(
                        "fixed-window",
                        map("limit", 1, "windowMs", 1000),
                        List.of(map("atMs", 0, "key", "alice"))),
                request(
                        "fixed-window",
                        map("limit", 1, "windowMs", 1000),
                        List.of(map("atMs", 0, "cost", 1))));
        assertEquals("invalid_request", errorCode(output.get(0)));
        assertEquals("invalid_request", errorCode(output.get(1)));
    }

    private static String errorCode(Map<String, Object> response) {
        return (String) object(response.get("error")).get("code");
    }

    private static List<Boolean> allowed(Map<String, Object> response) {
        List<Boolean> result = new ArrayList<>();
        for (Object raw : array(response.get("decisions"))) {
            result.add((Boolean) object(raw).get("allowed"));
        }
        return result;
    }

    private static List<Object> remaining(Map<String, Object> response) {
        List<Object> result = new ArrayList<>();
        for (Object raw : array(response.get("decisions"))) {
            result.add(object(raw).get("remaining"));
        }
        return result;
    }

    private static void assertNumbersEqual(List<Object> expected, List<Object> actual, String caseName) {
        assertEquals(expected.size(), actual.size(), caseName);
        for (int index = 0; index < expected.size(); index++) {
            assertEquals(
                    ((Number) expected.get(index)).doubleValue(),
                    ((Number) actual.get(index)).doubleValue(),
                    0.000001,
                    caseName + " remaining[" + index + "]");
        }
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> object(Object value) {
        return (Map<String, Object>) value;
    }

    @SuppressWarnings("unchecked")
    private static List<Object> array(Object value) {
        return (List<Object>) value;
    }

    private static Map<String, Object> map(Object... values) {
        Map<String, Object> result = new LinkedHashMap<>();
        for (int index = 0; index < values.length; index += 2) {
            result.put((String) values[index], values[index + 1]);
        }
        return result;
    }

    /** Independent test-only JSON codec: assertions observe the CLI response, not runner internals. */
    private static final class TestJson {
        static Object parse(String source) {
            return new Parser(source).parse();
        }

        static String stringify(Object value) {
            StringBuilder output = new StringBuilder();
            write(value, output);
            return output.toString();
        }

        private static void write(Object value, StringBuilder output) {
            if (value == null) {
                output.append("null");
            } else if (value instanceof String string) {
                output.append('"').append(string.replace("\\", "\\\\").replace("\"", "\\\"")).append('"');
            } else if (value instanceof Number || value instanceof Boolean) {
                output.append(value);
            } else if (value instanceof Map<?, ?> map) {
                output.append('{');
                boolean first = true;
                for (Map.Entry<?, ?> entry : map.entrySet()) {
                    if (!first) output.append(',');
                    first = false;
                    write(entry.getKey(), output);
                    output.append(':');
                    write(entry.getValue(), output);
                }
                output.append('}');
            } else if (value instanceof Iterable<?> iterable) {
                output.append('[');
                boolean first = true;
                for (Object item : iterable) {
                    if (!first) output.append(',');
                    first = false;
                    write(item, output);
                }
                output.append(']');
            } else {
                throw new IllegalArgumentException("unsupported JSON value");
            }
        }

        private static final class Parser {
            private final String source;
            private int cursor;

            Parser(String source) {
                this.source = source;
            }

            Object parse() {
                Object value = value();
                whitespace();
                if (cursor != source.length()) throw new IllegalArgumentException("trailing JSON");
                return value;
            }

            private Object value() {
                whitespace();
                return switch (source.charAt(cursor)) {
                    case '{' -> object();
                    case '[' -> array();
                    case '"' -> string();
                    case 't' -> literal("true", true);
                    case 'f' -> literal("false", false);
                    case 'n' -> literal("null", null);
                    default -> number();
                };
            }

            private Map<String, Object> object() {
                cursor++;
                Map<String, Object> result = new LinkedHashMap<>();
                whitespace();
                if (take('}')) return result;
                while (true) {
                    String key = string();
                    whitespace();
                    expect(':');
                    result.put(key, value());
                    whitespace();
                    if (take('}')) return result;
                    expect(',');
                    whitespace();
                }
            }

            private List<Object> array() {
                cursor++;
                List<Object> result = new ArrayList<>();
                whitespace();
                if (take(']')) return result;
                while (true) {
                    result.add(value());
                    whitespace();
                    if (take(']')) return result;
                    expect(',');
                }
            }

            private String string() {
                expect('"');
                StringBuilder result = new StringBuilder();
                while (true) {
                    char character = source.charAt(cursor++);
                    if (character == '"') return result.toString();
                    if (character != '\\') {
                        result.append(character);
                        continue;
                    }
                    char escape = source.charAt(cursor++);
                    switch (escape) {
                        case '"', '\\', '/' -> result.append(escape);
                        case 'b' -> result.append('\b');
                        case 'f' -> result.append('\f');
                        case 'n' -> result.append('\n');
                        case 'r' -> result.append('\r');
                        case 't' -> result.append('\t');
                        case 'u' -> {
                            result.append((char) Integer.parseInt(source.substring(cursor, cursor + 4), 16));
                            cursor += 4;
                        }
                        default -> throw new IllegalArgumentException("bad escape");
                    }
                }
            }

            private Number number() {
                int start = cursor;
                while (cursor < source.length()
                        && "-+0123456789.eE".indexOf(source.charAt(cursor)) >= 0) cursor++;
                double value = Double.parseDouble(source.substring(start, cursor));
                if (value == Math.rint(value)) return (long) value;
                return value;
            }

            private Object literal(String text, Object value) {
                cursor += text.length();
                return value;
            }

            private boolean take(char character) {
                if (cursor < source.length() && source.charAt(cursor) == character) {
                    cursor++;
                    return true;
                }
                return false;
            }

            private void expect(char character) {
                if (!take(character)) throw new IllegalArgumentException("expected " + character);
            }

            private void whitespace() {
                while (cursor < source.length() && Character.isWhitespace(source.charAt(cursor))) cursor++;
            }
        }
    }
}
