import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

final class PublisherSubscriberPattern {
    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String, Object> input) {
        List<Map<String, Object>> subscribers = (List<Map<String, Object>>) input.getOrDefault("subscribers", List.of());
        Set<String> seen = new HashSet<>();
        for (Map<String, Object> subscriber : subscribers) {
            if (!seen.add(String.valueOf(subscriber.get("name")))) throw new PatternException("duplicate_subscriber");
        }
        List<Map<String, Object>> events = (List<Map<String, Object>>) input.getOrDefault("events", List.of());
        List<Map<String, Object>> deliveries = new ArrayList<>();
        for (Map<String, Object> event : events) {
            for (Map<String, Object> subscriber : subscribers) {
                List<Object> topics = (List<Object>) subscriber.getOrDefault("topics", List.of());
                if (!topics.contains(event.get("topic"))) continue;
                String outcome = String.valueOf(subscriber.getOrDefault("outcome", "success"));
                if (!outcome.equals("success") && !outcome.equals("failure")) throw new PatternException("unknown_delivery_outcome");
                deliveries.add(Map.of("event_id", event.get("id"), "subscriber", subscriber.get("name"), "status", outcome.equals("success") ? "delivered" : "failed"));
            }
        }
        return Map.of("published", events.size(), "deliveries", deliveries);
    }
}
