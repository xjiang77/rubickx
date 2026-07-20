import java.util.ArrayList;
import java.util.List;
import java.util.Map;

final class TransactionalOutboxPattern {
    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String, Object> input) {
        List<Object> attempts = (List<Object>) input.getOrDefault("relay_attempts", List.of());
        if (!Boolean.TRUE.equals(input.get("commit"))) {
            if (!attempts.isEmpty()) throw new PatternException("relay_without_commit");
            return Map.of("aggregate_state", "absent", "outbox_status", "absent", "deliveries", List.of(), "relay_receipts", List.of());
        }
        String messageId = String.valueOf(input.get("message_id"));
        String status = "pending";
        List<String> deliveries = new ArrayList<>();
        List<String> receipts = new ArrayList<>();
        for (Object raw : attempts) {
            String attempt = String.valueOf(raw);
            if (status.equals("sent")) { receipts.add("skipped_already_sent:" + messageId); continue; }
            switch (attempt) {
                case "crash_before_publish" -> receipts.add("crash_before_publish:" + messageId);
                case "crash_after_publish" -> { deliveries.add(messageId); receipts.add("crash_after_publish:" + messageId); }
                case "success" -> { deliveries.add(messageId); receipts.add("published:" + messageId); status = "sent"; }
                default -> throw new PatternException("unknown_relay_outcome");
            }
        }
        return Map.of("aggregate_state", input.get("new_state"), "outbox_status", status, "deliveries", deliveries, "relay_receipts", receipts);
    }
}
