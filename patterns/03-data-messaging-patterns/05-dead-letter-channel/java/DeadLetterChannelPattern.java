import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class DeadLetterChannelPattern {
    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String, Object> input) {
        int budget = ((Number) input.get("max_attempts")).intValue();
        if (budget < 1) throw new PatternException("invalid_attempt_budget");
        List<String> processed = new ArrayList<>(), receipts = new ArrayList<>();
        List<Map<String, Object>> dead = new ArrayList<>();
        Map<String, Map<String, Object>> byId = new LinkedHashMap<>();
        for (Map<String, Object> message : (List<Map<String, Object>>) input.getOrDefault("messages", List.of())) {
            String id = String.valueOf(message.get("id"));
            if (byId.putIfAbsent(id, message) != null) throw new PatternException("duplicate_message_id");
            List<Object> outcomes = (List<Object>) message.getOrDefault("outcomes", List.of());
            if (outcomes.isEmpty()) throw new PatternException("missing_outcome");
            for (int index = 0; index < budget; index++) {
                String outcome = String.valueOf(outcomes.get(Math.min(index, outcomes.size()-1))); int attempt = index+1;
                if (outcome.equals("success")) { processed.add(id); receipts.add("processed:"+id+":"+attempt); break; }
                if (outcome.equals("permanent")) { dead.add(new LinkedHashMap<>(Map.of("id",id,"reason","permanent","attempts",attempt,"status","dead"))); receipts.add("dead:"+id+":permanent:"+attempt); break; }
                if (!outcome.equals("transient")) throw new PatternException("unknown_processing_outcome");
                if (attempt == budget) { dead.add(new LinkedHashMap<>(Map.of("id",id,"reason","transient_exhausted","attempts",attempt,"status","dead"))); receipts.add("dead:"+id+":transient_exhausted:"+attempt); }
                else receipts.add("retry:"+id+":"+attempt);
            }
        }
        for (Object raw : (List<Object>) input.getOrDefault("replay_ids", List.of())) {
            String id = String.valueOf(raw); Map<String,Object> record = dead.stream().filter(item -> item.get("id").equals(id) && item.get("status").equals("dead")).findFirst().orElseThrow(() -> new PatternException("replay_not_dead"));
            String outcome = String.valueOf(byId.get(id).getOrDefault("replay_outcome", "failure"));
            if (outcome.equals("success")) { record.put("status","replayed"); processed.add(id); receipts.add("replayed:"+id); }
            else if (outcome.equals("failure")) receipts.add("replay_failed:"+id);
            else throw new PatternException("unknown_replay_outcome");
        }
        return Map.of("processed", processed, "dead_letters", dead, "receipts", receipts);
    }
}
