import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * A dependency-free JSON Lines adapter for the five deterministic rate-limiter algorithms.
 *
 * <p>The process accepts one fixed-shape RunRequest per line and writes exactly one response per
 * line. Invalid input does not terminate the long-lived adapter.
 */
public final class RateLimiterRunner {
    private static final long MAX_SAFE_INTEGER_MS = 9_007_199_254_740_991L;

    private RateLimiterRunner() {}

    public static void main(String[] args) throws IOException {
        BufferedReader input = new BufferedReader(
                new InputStreamReader(System.in, StandardCharsets.UTF_8));
        PrintWriter output = new PrintWriter(System.out, true, StandardCharsets.UTF_8);
        String line;
        while ((line = input.readLine()) != null) {
            if (line.isBlank()) {
                continue;
            }
            output.println(Json.stringify(handleLine(line)));
        }
    }

    private static Map<String, Object> handleLine(String line) {
        Object request;
        try {
            request = Json.parse(line);
        } catch (Json.ParseFailure failure) {
            return error("invalid_json", "invalid JSON: " + failure.getMessage());
        }
        try {
            return runRequest(request);
        } catch (InvalidRequest failure) {
            return error("invalid_request", failure.getMessage());
        } catch (RuntimeException failure) {
            return error("internal_error", "runner failed: " + failure.getClass().getSimpleName());
        }
    }

    private static Map<String, Object> error(String code, String message) {
        return map("error", map("code", code, "message", message));
    }

    private static Map<String, Object> runRequest(Object rawRequest) {
        Map<String, Object> request = object(rawRequest, "request");
        Object rawAlgorithm = request.get("algorithm");
        if (!(rawAlgorithm instanceof String algorithm)) {
            throw new InvalidRequest("algorithm must be a string");
        }
        List<TimelineItem> timeline = validateTimeline(request.get("requestTimeline"));
        Observer observer = new Observer();
        List<Map<String, Object>> decisions = switch (algorithm) {
            case "fixed-window" -> runFixedWindow(request.get("config"), timeline, observer);
            case "sliding-window-log" -> runSlidingLog(request.get("config"), timeline, observer);
            case "sliding-window-counter" -> runSlidingCounter(request.get("config"), timeline, observer);
            case "token-bucket" -> runTokenBucket(request.get("config"), timeline, observer);
            case "leaky-bucket" -> runLeakyBucket(request.get("config"), timeline, observer);
            default -> throw new InvalidRequest("unsupported algorithm: " + algorithm);
        };
        return map("events", observer.events, "decisions", decisions);
    }

    private record TimelineItem(double atMs, double cost, String key) {}

    private static final class InvalidRequest extends RuntimeException {
        InvalidRequest(String message) {
            super(message);
        }
    }

    private static final class Observer {
        private final List<Map<String, Object>> events = new ArrayList<>();

        void emit(
                String stepId,
                TimelineItem item,
                Object before,
                Object after,
                Object decision,
                String reason) {
            events.add(map(
                    "seq", events.size() + 1L,
                    "stepId", stepId,
                    "actor", item.key(),
                    "timestampMs", rounded(item.atMs()),
                    "before", normalise(deepCopy(before)),
                    "after", normalise(deepCopy(after)),
                    "decision", normalise(deepCopy(decision)),
                    "reason", reason));
        }
    }

    private static List<TimelineItem> validateTimeline(Object rawTimeline) {
        if (!(rawTimeline instanceof List<?> values)) {
            throw new InvalidRequest("requestTimeline must be an array");
        }
        if (values.size() > 100) {
            throw new InvalidRequest("requestTimeline must contain at most 100 items");
        }
        List<TimelineItem> result = new ArrayList<>();
        double previous = -1;
        for (int index = 0; index < values.size(); index++) {
            Map<String, Object> value = object(values.get(index), "requestTimeline[" + index + "]");
            double atMs = number(value.get("atMs"), "requestTimeline[" + index + "].atMs", false);
            double cost = quantity(
                    value.get("cost"),
                    "requestTimeline[" + index + "].cost");
            Object rawKey = value.get("key");
            if (atMs != Math.rint(atMs)
                    || atMs < 0
                    || atMs > MAX_SAFE_INTEGER_MS
                    || atMs < previous) {
                throw new InvalidRequest(
                        "requestTimeline atMs must be non-negative, non-decreasing safe integer milliseconds");
            }
            if (!(rawKey instanceof String key)
                    || key.isEmpty()
                    || key.getBytes(StandardCharsets.UTF_8).length > 128) {
                throw new InvalidRequest(
                        "requestTimeline[" + index + "].key must be a non-empty UTF-8 string "
                        + "of at most 128 bytes");
            }
            result.add(new TimelineItem(atMs, cost, key));
            previous = atMs;
        }
        return result;
    }

    private static double[] windowConfig(Object rawConfig) {
        Map<String, Object> config = object(rawConfig, "config");
        double limit = quantity(config.get("limit"), "config.limit");
        double windowMs = number(config.get("windowMs"), "config.windowMs", true);
        if (windowMs != Math.rint(windowMs) || windowMs > MAX_SAFE_INTEGER_MS) {
            throw new InvalidRequest(
                    "config.windowMs must be a positive safe integer in milliseconds");
        }
        return new double[] {limit, windowMs};
    }

    private static double[] bucketConfig(Object rawConfig) {
        Map<String, Object> config = object(rawConfig, "config");
        double capacity = quantity(config.get("capacity"), "config.capacity");
        double rate = quantity(config.get("ratePerSecond"), "config.ratePerSecond");
        double maximumRecoveryMs = capacity / (rate / 1000);
        if (!Double.isFinite(maximumRecoveryMs) || maximumRecoveryMs > MAX_SAFE_INTEGER_MS) {
            throw new InvalidRequest(
                    "config capacity and ratePerSecond must yield a finite maximum recovery "
                    + "time no greater than " + MAX_SAFE_INTEGER_MS + " milliseconds");
        }
        return new double[] {capacity, rate};
    }

    private static List<Map<String, Object>> runFixedWindow(
            Object config, List<TimelineItem> timeline, Observer observer) {
        double[] values = windowConfig(config);
        double limit = values[0];
        double windowMs = values[1];
        Map<String, Map<String, Object>> states = new LinkedHashMap<>();
        List<Map<String, Object>> decisions = new ArrayList<>();
        for (TimelineItem item : timeline) {
            double windowStartMs = Math.floor(item.atMs() / windowMs) * windowMs;
            Map<String, Object> state = states.computeIfAbsent(
                    item.key(), ignored -> map("windowStartMs", windowStartMs, "count", 0.0));
            Object before = deepCopy(state);
            // @step:fixed.locate-window
            if (value(state, "windowStartMs") != windowStartMs) {
                state.put("windowStartMs", windowStartMs);
                state.put("count", 0.0);
            }
            observer.emit("fixed.locate-window", item, before, state, null, "window_selected");

            Object decisionBefore = deepCopy(state);
            boolean allowed = value(state, "count") + item.cost() <= limit;
            if (allowed) {
                state.put("count", value(state, "count") + item.cost());
            }
            String reason = allowed
                    ? "within_limit"
                    : (item.cost() > limit ? "cost_exceeds_limit" : "limit_exceeded");
            double resetAtMs = windowStartMs + windowMs;
            // @step:fixed.decision
            Map<String, Object> decision = decision(
                    allowed,
                    limit - value(state, "count"),
                    allowed || item.cost() > limit ? 0 : resetAtMs - item.atMs(),
                    resetAtMs,
                    reason);
            observer.emit("fixed.decision", item, decisionBefore, state, decision, reason);
            decisions.add(decision);
        }
        return decisions;
    }

    private static List<Map<String, Object>> runSlidingLog(
            Object config, List<TimelineItem> timeline, Observer observer) {
        double[] values = windowConfig(config);
        double limit = values[0];
        double windowMs = values[1];
        Map<String, List<Map<String, Object>>> states = new LinkedHashMap<>();
        List<Map<String, Object>> decisions = new ArrayList<>();
        for (TimelineItem item : timeline) {
            List<Map<String, Object>> entries = states.computeIfAbsent(item.key(), ignored -> new ArrayList<>());
            double usedBefore = entries.stream().mapToDouble(entry -> value(entry, "cost")).sum();
            Object before = map("entries", deepCopy(entries), "used", usedBefore);
            double cutoff = item.atMs() - windowMs;
            // @step:sliding-log.evict
            entries.removeIf(entry -> value(entry, "atMs") <= cutoff);
            double used = entries.stream().mapToDouble(entry -> value(entry, "cost")).sum();
            Object afterEvict = map("entries", deepCopy(entries), "used", used);
            observer.emit("sliding-log.evict", item, before, afterEvict, null, "expired_entries_removed");

            boolean allowed = used + item.cost() <= limit;
            if (allowed) {
                entries.add(map("atMs", item.atMs(), "cost", item.cost()));
                used += item.cost();
            }
            String reason = allowed
                    ? "within_limit"
                    : (item.cost() > limit ? "cost_exceeds_limit" : "limit_exceeded");
            double retryAfterMs = 0;
            if (!allowed && item.cost() <= limit) {
                double requiredRelease = used + item.cost() - limit;
                double released = 0;
                for (Map<String, Object> entry : entries) {
                    released += value(entry, "cost");
                    if (released + 1e-9 >= requiredRelease) {
                        retryAfterMs = Math.max(0, value(entry, "atMs") + windowMs - item.atMs());
                        break;
                    }
                }
            }
            double resetAtMs = allowed
                    ? (entries.isEmpty() ? item.atMs() : value(entries.get(0), "atMs") + windowMs)
                    : item.atMs() + retryAfterMs;
            Object after = map("entries", deepCopy(entries), "used", used);
            // @step:sliding-log.decision
            Map<String, Object> decision = decision(
                    allowed, limit - used, retryAfterMs, resetAtMs, reason);
            observer.emit("sliding-log.decision", item, afterEvict, after, decision, reason);
            decisions.add(decision);
        }
        return decisions;
    }

    private static List<Map<String, Object>> runSlidingCounter(
            Object config, List<TimelineItem> timeline, Observer observer) {
        double[] values = windowConfig(config);
        double limit = values[0];
        double windowMs = values[1];
        Map<String, Map<String, Object>> states = new LinkedHashMap<>();
        List<Map<String, Object>> decisions = new ArrayList<>();
        for (TimelineItem item : timeline) {
            double currentWindowStartMs = Math.floor(item.atMs() / windowMs) * windowMs;
            Map<String, Object> state = states.computeIfAbsent(
                    item.key(),
                    ignored -> map(
                            "currentWindowStartMs", currentWindowStartMs,
                            "currentCount", 0.0,
                            "previousCount", 0.0));
            Object before = deepCopy(state);
            long windowsElapsed = (long) ((currentWindowStartMs - value(state, "currentWindowStartMs")) / windowMs);
            // @step:sliding-counter.rotate
            if (windowsElapsed == 1) {
                state.put("previousCount", value(state, "currentCount"));
                state.put("currentCount", 0.0);
                state.put("currentWindowStartMs", currentWindowStartMs);
            } else if (windowsElapsed > 1) {
                state.put("previousCount", 0.0);
                state.put("currentCount", 0.0);
                state.put("currentWindowStartMs", currentWindowStartMs);
            }
            observer.emit("sliding-counter.rotate", item, before, state, null, "windows_rotated");

            double elapsed = item.atMs() - currentWindowStartMs;
            double previousWeight = Math.max(0, 1 - elapsed / windowMs);
            // @step:sliding-counter.estimate
            double estimatedCount = value(state, "currentCount")
                    + value(state, "previousCount") * previousWeight;
            Object estimateState = with(state, "previousWeight", previousWeight, "estimatedCount", estimatedCount);
            observer.emit(
                    "sliding-counter.estimate",
                    item,
                    state,
                    estimateState,
                    null,
                    "weighted_count_estimated");

            boolean allowed = estimatedCount + item.cost() <= limit;
            if (allowed) {
                state.put("currentCount", value(state, "currentCount") + item.cost());
                estimatedCount += item.cost();
            }
            String reason = allowed
                    ? "within_limit"
                    : (item.cost() > limit ? "cost_exceeds_limit" : "limit_exceeded");
            double resetAtMs = currentWindowStartMs + windowMs;
            double retryAfterMs = 0;
            if (!allowed && item.cost() <= limit) {
                double excess = Math.max(0, estimatedCount + item.cost() - limit);
                double untilBoundary = Math.max(0, currentWindowStartMs + windowMs - item.atMs());
                double previousCount = value(state, "previousCount");
                double currentCount = value(state, "currentCount");
                if (currentCount + item.cost() <= limit && previousCount > 0) {
                    retryAfterMs = Math.min(untilBoundary, excess * windowMs / previousCount);
                } else {
                    retryAfterMs = untilBoundary;
                    if (currentCount > 0) {
                        retryAfterMs += Math.max(
                                0,
                                (currentCount + item.cost() - limit) * windowMs / currentCount);
                    }
                }
                resetAtMs = item.atMs() + retryAfterMs;
            }
            Object after = with(state, "previousWeight", previousWeight, "estimatedCount", estimatedCount);
            // @step:sliding-counter.decision
            Map<String, Object> decision = decision(
                    allowed,
                    limit - estimatedCount,
                    retryAfterMs,
                    resetAtMs,
                    reason);
            observer.emit("sliding-counter.decision", item, estimateState, after, decision, reason);
            decisions.add(decision);
        }
        return decisions;
    }

    private static List<Map<String, Object>> runTokenBucket(
            Object config, List<TimelineItem> timeline, Observer observer) {
        double[] values = bucketConfig(config);
        double capacity = values[0];
        double rate = values[1];
        Map<String, Map<String, Object>> states = new LinkedHashMap<>();
        List<Map<String, Object>> decisions = new ArrayList<>();
        for (TimelineItem item : timeline) {
            Map<String, Object> state = states.computeIfAbsent(
                    item.key(), ignored -> map("tokens", capacity, "lastRefillMs", item.atMs()));
            Object before = deepCopy(state);
            double elapsed = Math.max(0, item.atMs() - value(state, "lastRefillMs"));
            // @step:token.refill
            state.put("tokens", Math.min(capacity, value(state, "tokens") + elapsed * rate / 1000));
            state.put("lastRefillMs", item.atMs());
            observer.emit("token.refill", item, before, state, null, "tokens_refilled");

            Object decisionBefore = deepCopy(state);
            boolean allowed = value(state, "tokens") + 1e-9 >= item.cost();
            if (allowed) {
                state.put("tokens", value(state, "tokens") - item.cost());
            }
            String reason = allowed
                    ? "token_available"
                    : (item.cost() > capacity ? "cost_exceeds_capacity" : "insufficient_tokens");
            double retryAfterMs = allowed || item.cost() > capacity
                    ? 0
                    : (item.cost() - value(state, "tokens")) / rate * 1000;
            double resetAtMs = item.atMs() + (capacity - value(state, "tokens")) / rate * 1000;
            // @step:token.decision
            Map<String, Object> decision = decision(
                    allowed, value(state, "tokens"), retryAfterMs, resetAtMs, reason);
            observer.emit("token.decision", item, decisionBefore, state, decision, reason);
            decisions.add(decision);
        }
        return decisions;
    }

    private static List<Map<String, Object>> runLeakyBucket(
            Object config, List<TimelineItem> timeline, Observer observer) {
        double[] values = bucketConfig(config);
        double capacity = values[0];
        double rate = values[1];
        Map<String, Map<String, Object>> states = new LinkedHashMap<>();
        List<Map<String, Object>> decisions = new ArrayList<>();
        for (TimelineItem item : timeline) {
            Map<String, Object> state = states.computeIfAbsent(
                    item.key(), ignored -> map("water", 0.0, "lastLeakMs", item.atMs()));
            Object before = deepCopy(state);
            double elapsed = Math.max(0, item.atMs() - value(state, "lastLeakMs"));
            // @step:leaky.drain
            state.put("water", Math.max(0, value(state, "water") - elapsed * rate / 1000));
            state.put("lastLeakMs", item.atMs());
            observer.emit("leaky.drain", item, before, state, null, "queued_work_drained");

            Object decisionBefore = deepCopy(state);
            boolean allowed = value(state, "water") + item.cost() <= capacity + 1e-9;
            if (allowed) {
                state.put("water", value(state, "water") + item.cost());
            }
            String reason = allowed
                    ? "queue_has_capacity"
                    : (item.cost() > capacity ? "cost_exceeds_capacity" : "queue_full");
            double retryAfterMs = allowed || item.cost() > capacity
                    ? 0
                    : (value(state, "water") + item.cost() - capacity) / rate * 1000;
            double resetAtMs = item.atMs() + value(state, "water") / rate * 1000;
            // @step:leaky.decision
            Map<String, Object> decision = decision(
                    allowed, capacity - value(state, "water"), retryAfterMs, resetAtMs, reason);
            observer.emit("leaky.decision", item, decisionBefore, state, decision, reason);
            decisions.add(decision);
        }
        return decisions;
    }

    private static Map<String, Object> decision(
            boolean allowed, double remaining, double retryAfterMs, double resetAtMs, String reason) {
        return map(
                "allowed", allowed,
                "remaining", rounded(Math.max(0, remaining)),
                "retryAfterMs", rounded(Math.max(0, retryAfterMs)),
                "resetAtMs", rounded(Math.max(0, resetAtMs)),
                "reason", reason);
    }

    private static double number(Object value, String name, boolean positive) {
        if (!(value instanceof Number number)) {
            throw new InvalidRequest(name + " must be a number");
        }
        double result = number.doubleValue();
        if (!Double.isFinite(result) || (positive && result <= 0)) {
            throw new InvalidRequest(name + " must be "
                    + (positive ? "a positive finite number" : "finite"));
        }
        return result;
    }

    private static double quantity(Object value, String name) {
        double result = number(value, name, true);
        if (result > MAX_SAFE_INTEGER_MS) {
            throw new InvalidRequest(
                    name + " must be a positive finite number no greater than "
                    + MAX_SAFE_INTEGER_MS);
        }
        return result;
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> object(Object value, String name) {
        if (!(value instanceof Map<?, ?> raw)) {
            throw new InvalidRequest(name + " must be an object");
        }
        for (Object key : raw.keySet()) {
            if (!(key instanceof String)) {
                throw new InvalidRequest(name + " keys must be strings");
            }
        }
        return (Map<String, Object>) raw;
    }

    private static double value(Map<String, Object> state, String key) {
        return ((Number) state.get(key)).doubleValue();
    }

    private static Map<String, Object> with(Map<String, Object> source, Object... values) {
        Map<String, Object> result = new LinkedHashMap<>(source);
        for (int index = 0; index < values.length; index += 2) {
            result.put((String) values[index], values[index + 1]);
        }
        return result;
    }

    private static Map<String, Object> map(Object... values) {
        Map<String, Object> result = new LinkedHashMap<>();
        for (int index = 0; index < values.length; index += 2) {
            result.put((String) values[index], values[index + 1]);
        }
        return result;
    }

    private static Number rounded(double value) {
        // Quantize the actual IEEE-754 value; exact binary ties go away from zero.
        double magnitude = Math.floor(Math.abs(value) * 1_000_000 + 0.5) / 1_000_000;
        double result = Math.copySign(magnitude, value);
        if (result == 0) {
            return 0L;
        }
        if (result == Math.rint(result) && result >= Long.MIN_VALUE && result <= Long.MAX_VALUE) {
            return (long) result;
        }
        return result;
    }

    private static Object normalise(Object value) {
        if (value instanceof Double number) {
            return rounded(number);
        }
        if (value instanceof Float number) {
            return rounded(number.doubleValue());
        }
        if (value instanceof List<?> list) {
            List<Object> result = new ArrayList<>();
            for (Object item : list) {
                result.add(normalise(item));
            }
            return result;
        }
        if (value instanceof Map<?, ?> raw) {
            Map<String, Object> result = new LinkedHashMap<>();
            for (Map.Entry<?, ?> entry : raw.entrySet()) {
                result.put((String) entry.getKey(), normalise(entry.getValue()));
            }
            return result;
        }
        return value;
    }

    private static Object deepCopy(Object value) {
        if (value instanceof List<?> list) {
            List<Object> result = new ArrayList<>();
            for (Object item : list) {
                result.add(deepCopy(item));
            }
            return result;
        }
        if (value instanceof Map<?, ?> raw) {
            Map<String, Object> result = new LinkedHashMap<>();
            for (Map.Entry<?, ?> entry : raw.entrySet()) {
                result.put((String) entry.getKey(), deepCopy(entry.getValue()));
            }
            return result;
        }
        return value;
    }

    /** Small standards-compliant JSON codec, used only at the process boundary. */
    private static final class Json {
        private Json() {}

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
                writeString(string, output);
            } else if (value instanceof Boolean || value instanceof Number) {
                output.append(value);
            } else if (value instanceof Map<?, ?> map) {
                output.append('{');
                boolean first = true;
                for (Map.Entry<?, ?> entry : map.entrySet()) {
                    if (!first) output.append(',');
                    first = false;
                    writeString((String) entry.getKey(), output);
                    output.append(':');
                    write(entry.getValue(), output);
                }
                output.append('}');
            } else if (value instanceof Iterable<?> values) {
                output.append('[');
                boolean first = true;
                for (Object item : values) {
                    if (!first) output.append(',');
                    first = false;
                    write(item, output);
                }
                output.append(']');
            } else {
                throw new IllegalArgumentException("cannot encode " + value.getClass());
            }
        }

        private static void writeString(String value, StringBuilder output) {
            output.append('"');
            for (int index = 0; index < value.length(); index++) {
                char character = value.charAt(index);
                switch (character) {
                    case '"' -> output.append("\\\"");
                    case '\\' -> output.append("\\\\");
                    case '\b' -> output.append("\\b");
                    case '\f' -> output.append("\\f");
                    case '\n' -> output.append("\\n");
                    case '\r' -> output.append("\\r");
                    case '\t' -> output.append("\\t");
                    default -> {
                        if (character < 0x20) {
                            output.append(String.format("\\u%04x", (int) character));
                        } else {
                            output.append(character);
                        }
                    }
                }
            }
            output.append('"');
        }

        private static final class ParseFailure extends RuntimeException {
            ParseFailure(String message) {
                super(message);
            }
        }

        private static final class Parser {
            private final String source;
            private int cursor;

            Parser(String source) {
                this.source = source;
            }

            Object parse() {
                Object value = parseValue();
                whitespace();
                if (cursor != source.length()) fail("unexpected trailing data");
                return value;
            }

            private Object parseValue() {
                whitespace();
                if (cursor >= source.length()) fail("unexpected end of input");
                return switch (source.charAt(cursor)) {
                    case '{' -> parseObject();
                    case '[' -> parseArray();
                    case '"' -> parseString();
                    case 't' -> literal("true", Boolean.TRUE);
                    case 'f' -> literal("false", Boolean.FALSE);
                    case 'n' -> literal("null", null);
                    default -> parseNumber();
                };
            }

            private Map<String, Object> parseObject() {
                cursor++;
                Map<String, Object> result = new LinkedHashMap<>();
                whitespace();
                if (take('}')) return result;
                while (true) {
                    whitespace();
                    if (cursor >= source.length() || source.charAt(cursor) != '"') fail("expected object key");
                    String key = parseString();
                    whitespace();
                    expect(':');
                    result.put(key, parseValue());
                    whitespace();
                    if (take('}')) return result;
                    expect(',');
                }
            }

            private List<Object> parseArray() {
                cursor++;
                List<Object> result = new ArrayList<>();
                whitespace();
                if (take(']')) return result;
                while (true) {
                    result.add(parseValue());
                    whitespace();
                    if (take(']')) return result;
                    expect(',');
                }
            }

            private String parseString() {
                expect('"');
                StringBuilder result = new StringBuilder();
                while (cursor < source.length()) {
                    char character = source.charAt(cursor++);
                    if (character == '"') return result.toString();
                    if (character == '\\') {
                        if (cursor >= source.length()) fail("unfinished escape");
                        char escape = source.charAt(cursor++);
                        switch (escape) {
                            case '"', '\\', '/' -> result.append(escape);
                            case 'b' -> result.append('\b');
                            case 'f' -> result.append('\f');
                            case 'n' -> result.append('\n');
                            case 'r' -> result.append('\r');
                            case 't' -> result.append('\t');
                            case 'u' -> {
                                if (cursor + 4 > source.length()) fail("unfinished unicode escape");
                                try {
                                    result.append((char) Integer.parseInt(source.substring(cursor, cursor + 4), 16));
                                } catch (NumberFormatException failure) {
                                    fail("invalid unicode escape");
                                }
                                cursor += 4;
                            }
                            default -> fail("invalid escape");
                        }
                    } else {
                        if (character < 0x20) fail("control character in string");
                        result.append(character);
                    }
                }
                fail("unterminated string");
                return null;
            }

            private Double parseNumber() {
                int start = cursor;
                if (take('-')) {}
                if (take('0')) {
                    // A leading zero is complete unless followed by a fraction or exponent.
                } else {
                    digits();
                }
                if (take('.')) digits();
                if (cursor < source.length() && (source.charAt(cursor) == 'e' || source.charAt(cursor) == 'E')) {
                    cursor++;
                    if (cursor < source.length() && (source.charAt(cursor) == '+' || source.charAt(cursor) == '-')) cursor++;
                    digits();
                }
                try {
                    return Double.valueOf(source.substring(start, cursor));
                } catch (NumberFormatException failure) {
                    fail("invalid number");
                    return null;
                }
            }

            private void digits() {
                int start = cursor;
                while (cursor < source.length() && Character.isDigit(source.charAt(cursor))) cursor++;
                if (start == cursor) fail("expected digit");
            }

            private Object literal(String text, Object value) {
                if (!source.startsWith(text, cursor)) fail("invalid literal");
                cursor += text.length();
                return value;
            }

            private boolean take(char expected) {
                if (cursor < source.length() && source.charAt(cursor) == expected) {
                    cursor++;
                    return true;
                }
                return false;
            }

            private void expect(char expected) {
                if (!take(expected)) fail("expected '" + expected + "'");
            }

            private void whitespace() {
                while (cursor < source.length() && Character.isWhitespace(source.charAt(cursor))) cursor++;
            }

            private void fail(String message) {
                throw new ParseFailure(message + " at offset " + cursor);
            }
        }
    }
}
