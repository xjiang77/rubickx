import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.fail;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.lang.reflect.Method;
import java.nio.file.Path;
import java.util.Map;

final class ContractHarness {
    @FunctionalInterface
    interface Evaluator {
        Object evaluate(Map<String, Object> input);
    }

    private static final ObjectMapper MAPPER = new ObjectMapper();

    private ContractHarness() {}

    static void run(String contractPath, Evaluator evaluator) {
        try {
            JsonNode contract = MAPPER.readTree(Path.of(contractPath).toFile());
            for (JsonNode scenario : contract.path("cases")) {
                String id = scenario.path("id").asText();
                Map<String, Object> input = MAPPER.convertValue(
                        scenario.path("input"), new TypeReference<>() {});
                try {
                    Object result = evaluator.evaluate(input);
                    if (scenario.has("expected_error")) {
                        fail(id + ": expected error "
                                + scenario.path("expected_error").path("code").asText());
                    }
                    assertEquals(scenario.path("expected"), MAPPER.valueToTree(result), id);
                } catch (RuntimeException error) {
                    if (!scenario.has("expected_error")) {
                        throw error;
                    }
                    assertEquals(
                            scenario.path("expected_error").path("code").asText(),
                            errorCode(error),
                            id);
                }
            }
        } catch (RuntimeException error) {
            throw error;
        } catch (Exception error) {
            throw new RuntimeException(error);
        }
    }

    private static String errorCode(RuntimeException error) {
        try {
            Method code = error.getClass().getDeclaredMethod("code");
            code.setAccessible(true);
            return String.valueOf(code.invoke(error));
        } catch (ReflectiveOperationException reflectionError) {
            throw new AssertionError(
                    "exception does not expose code(): " + error.getClass().getName(),
                    reflectionError);
        }
    }
}

