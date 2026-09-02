import java.util.ArrayList;
import java.util.List;
import java.util.Map;

final class SagaPattern {
    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }

    private static List<String> names(List<Map<String, Object>> steps) {
        return steps.stream().map(step -> String.valueOf(step.get("name"))).toList();
    }

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String, Object> input) {
        List<Map<String, Object>> completed = new ArrayList<>();
        List<String> actions = new ArrayList<>();
        for (Map<String, Object> step : (List<Map<String, Object>>) input.getOrDefault("steps", List.of())) {
            String name = String.valueOf(step.get("name"));
            actions.add("execute:" + name);
            String result = String.valueOf(step.get("result"));
            if (result.equals("success")) { completed.add(step); continue; }
            if (!result.equals("failure")) throw new PatternException("unknown_step_result");
            actions.add("failed:" + name);
            List<String> failed = new ArrayList<>();
            for (int i = completed.size() - 1; i >= 0; i--) {
                Map<String, Object> prior = completed.get(i);
                String outcome = String.valueOf(prior.getOrDefault("compensation", "success"));
                if (!outcome.equals("success") && !outcome.equals("failure")) throw new PatternException("unknown_compensation_result");
                String priorName = String.valueOf(prior.get("name"));
                actions.add("compensate:" + priorName + ":" + outcome);
                if (outcome.equals("failure")) failed.add(priorName);
            }
            return Map.of("status", failed.isEmpty() ? "compensated" : "recovery_required", "completed", names(completed), "failed_step", name, "failed_compensations", failed, "actions", actions);
        }
        return Map.of("status", "completed", "completed", names(completed), "failed_step", "none", "failed_compensations", List.of(), "actions", actions);
    }
}
