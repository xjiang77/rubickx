import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

final class CqrsPattern {
    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }
    record Change(int delta) {}

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String, Object> input) {
        List<Change> changes = new ArrayList<>(); Set<String> seen = new HashSet<>(); int writeBalance = 0;
        for (Map<String,Object> command : (List<Map<String,Object>>) input.getOrDefault("commands", List.of())) {
            String id = String.valueOf(command.get("id")); if (!seen.add(id)) throw new PatternException("duplicate_command_id");
            int delta = ((Number) command.get("delta")).intValue(); writeBalance += delta; changes.add(new Change(delta));
        }
        int projectionBalance = 0, projectionVersion = 0; List<Map<String,Object>> snapshots = new ArrayList<>();
        for (Object raw : (List<Object>) input.getOrDefault("projection_targets", List.of())) {
            int target = ((Number) raw).intValue(); if (target < projectionVersion) throw new PatternException("projection_regression"); if (target > changes.size()) throw new PatternException("projection_ahead");
            for (int index=projectionVersion; index<target; index++) projectionBalance += changes.get(index).delta();
            projectionVersion = target; snapshots.add(Map.of("balance", projectionBalance, "version", projectionVersion, "lag", changes.size()-projectionVersion));
        }
        return Map.of("write_model", Map.of("balance", writeBalance, "version", changes.size()), "projection_snapshots", snapshots);
    }
}
