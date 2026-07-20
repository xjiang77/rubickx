import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

public final class ChatAdapter {
    private ChatAdapter() {}

    public record Message(String role, String content) {}

    public record ChatRequest(String model, List<Message> messages, boolean requiresTools) {}

    public record ChatResponse(String content, String finishReason) {}

    public record LegacyRequest(String deployment, String prompt) {}

    public record LegacyResponse(String output, String stopCode) {}

    public interface LegacyClient {
        LegacyResponse generate(LegacyRequest request);
    }

    public static final class ProviderException extends RuntimeException {
        private final String code;

        public ProviderException(String code, String message) {
            super(message);
            this.code = code;
        }

        public String code() {
            return code;
        }
    }

    public static final class NormalizedException extends RuntimeException {
        private final String code;
        private final boolean retryable;

        public NormalizedException(String code, boolean retryable, String message) {
            super(message);
            this.code = code;
            this.retryable = retryable;
        }

        public String code() {
            return code;
        }

        public boolean retryable() {
            return retryable;
        }
    }

    public static final class LegacyProviderAdapter {
        private static final Map<String, String> MODEL_MAP =
                Map.of("chat-pro", "legacy-chat-pro");
        private static final Map<String, String> STOP_MAP =
                Map.of("stop", "stop", "length", "length");

        private final LegacyClient client;

        public LegacyProviderAdapter(LegacyClient client) {
            this.client = client;
        }

        public ChatResponse complete(ChatRequest request) {
            if (request.requiresTools()) {
                throw new NormalizedException(
                        "unsupported_capability",
                        false,
                        "legacy provider does not support tools");
            }

            String deployment = MODEL_MAP.get(request.model());
            if (deployment == null) {
                throw new NormalizedException(
                        "invalid_request",
                        false,
                        "unknown model: " + request.model());
            }

            String prompt = request.messages().stream()
                    .map(message -> message.role() + ": " + message.content())
                    .collect(Collectors.joining("\n"));

            final LegacyResponse response;
            try {
                response = client.generate(new LegacyRequest(deployment, prompt));
            } catch (ProviderException error) {
                throw normalizeError(error);
            }

            return new ChatResponse(
                    response.output(),
                    STOP_MAP.getOrDefault(response.stopCode(), "unknown"));
        }

        private static NormalizedException normalizeError(ProviderException error) {
            return switch (error.code()) {
                case "OVER_QUOTA" ->
                        new NormalizedException("rate_limited", true, error.getMessage());
                case "BAD_REQUEST" ->
                        new NormalizedException("invalid_request", false, error.getMessage());
                default ->
                        new NormalizedException("upstream_failure", true, error.getMessage());
            };
        }
    }
}
