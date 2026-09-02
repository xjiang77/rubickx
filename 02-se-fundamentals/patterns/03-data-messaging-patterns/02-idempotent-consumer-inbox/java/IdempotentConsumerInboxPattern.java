import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class IdempotentConsumerInboxPattern {
    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String, Object> input) {
        Map<String, Integer> inbox = new LinkedHashMap<>();
        List<String> receipts = new ArrayList<>();
        int balance = 0;
        for (Map<String, Object> message : (List<Map<String, Object>>) input.getOrDefault("messages", List.of())) {
            String id = String.valueOf(message.get("id"));
            int amount = ((Number) message.get("amount")).intValue();
            if (inbox.containsKey(id)) {
                if (inbox.get(id) != amount) throw new PatternException("message_identity_conflict");
                receipts.add("duplicate:" + id);
            } else {
                inbox.put(id, amount);
                balance += amount;
                receipts.add("applied:" + id);
            }
        }
        return Map.of("receipts", receipts, "balance", balance, "inbox_ids", new ArrayList<>(inbox.keySet()));
    }
}
