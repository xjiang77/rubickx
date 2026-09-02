import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Deque;
import java.util.List;
import java.util.Map;

final class BoundedProducerConsumerPattern {
    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String,Object> input) {
        int capacity = ((Number) input.get("capacity")).intValue(); if (capacity < 0) throw new PatternException("invalid_capacity");
        Deque<String> queue = new ArrayDeque<>(); List<String> consumed = new ArrayList<>(), receipts = new ArrayList<>(); boolean closed = false;
        for (Map<String,Object> action : (List<Map<String,Object>>) input.getOrDefault("actions", List.of())) {
            String operation = String.valueOf(action.get("op"));
            switch (operation) {
                case "produce" -> { String item = String.valueOf(action.get("item")); if (closed) throw new PatternException("produce_after_close"); if (queue.size() >= capacity) receipts.add("backpressured:"+item); else { queue.addLast(item); receipts.add("queued:"+item); } }
                case "consume" -> { if (!queue.isEmpty()) { String item=queue.removeFirst(); consumed.add(item); receipts.add("consumed:"+item); } else receipts.add(closed ? "end" : "empty"); }
                case "close" -> { if (closed) throw new PatternException("already_closed"); closed=true; receipts.add("closed"); }
                default -> throw new PatternException("unknown_action");
            }
        }
        return Map.of("receipts",receipts,"consumed",consumed,"remaining",new ArrayList<>(queue),"closed",closed);
    }
}
