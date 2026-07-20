import java.util.ArrayList;
import java.util.List;
import java.util.Map;

final class EventSourcingPattern {
    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }

    private static int applyEvent(int balance, Map<String,Object> event) {
        int amount = ((Number) event.get("amount")).intValue(); if (amount <= 0) throw new PatternException("invalid_event_amount");
        return switch (String.valueOf(event.get("type"))) {
            case "deposited" -> balance + amount;
            case "withdrawn" -> { if (amount > balance) throw new PatternException("invalid_event_history"); yield balance - amount; }
            default -> throw new PatternException("unknown_event_type");
        };
    }

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String,Object> input) {
        List<Map<String,Object>> events = (List<Map<String,Object>>) input.getOrDefault("events", List.of()); int balance = 0;
        for (Map<String,Object> event : events) balance = applyEvent(balance, event);
        List<Map<String,Object>> appended = new ArrayList<>();
        if (input.containsKey("command")) {
            Map<String,Object> command = (Map<String,Object>) input.get("command");
            if (((Number) command.get("expected_version")).intValue() != events.size()) throw new PatternException("version_conflict");
            int amount = ((Number) command.get("amount")).intValue(); if (amount <= 0) throw new PatternException("invalid_command_amount");
            String eventType = switch (String.valueOf(command.get("type"))) { case "deposit" -> "deposited"; case "withdraw" -> { if (amount > balance) throw new PatternException("insufficient_funds"); yield "withdrawn"; } default -> throw new PatternException("unknown_command_type"); };
            Map<String,Object> event = Map.of("type",eventType,"amount",amount,"version",events.size()+1); balance = applyEvent(balance,event); appended.add(event);
        }
        int count = events.size()+appended.size(); return Map.of("balance",balance,"version",count,"history_count",count,"appended_events",appended);
    }
}
