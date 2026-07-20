import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import org.junit.jupiter.api.Test;

class ChatAdapterTest {
    private static final JsonNode CONTRACT = readContract();

    private static final class FakeLegacyClient implements ChatAdapter.LegacyClient {
        private ChatAdapter.LegacyResponse response;
        private RuntimeException error;
        private ChatAdapter.LegacyRequest lastRequest;
        private int calls;

        @Override
        public ChatAdapter.LegacyResponse generate(ChatAdapter.LegacyRequest request) {
            calls++;
            lastRequest = request;
            if (error != null) {
                throw error;
            }
            return response;
        }
    }

    @Test
    void mapsSharedContractFixture() {
        JsonNode scenario = contractCase("maps_request_and_response");
        JsonNode provider = scenario.path("input").path("provider_response");
        FakeLegacyClient client = new FakeLegacyClient();
        client.response = new ChatAdapter.LegacyResponse(
                provider.path("output").asText(),
                provider.path("stop_code").asText());

        ChatAdapter.ChatResponse response =
                new ChatAdapter.LegacyProviderAdapter(client).complete(requestFrom(scenario));

        JsonNode mapped = scenario.path("expected").path("mapped_request");
        JsonNode expectedResponse = scenario.path("expected").path("response");
        assertEquals(mapped.path("deployment").asText(), client.lastRequest.deployment());
        assertEquals(mapped.path("prompt").asText(), client.lastRequest.prompt());
        assertEquals(expectedResponse.path("content").asText(), response.content());
        assertEquals(expectedResponse.path("finish_reason").asText(), response.finishReason());
    }

    @Test
    void failsExplicitlyForUnknownModel() {
        JsonNode scenario = contractCase("rejects_unknown_model");
        ChatAdapter.NormalizedException error = assertThrows(
                ChatAdapter.NormalizedException.class,
                () -> new ChatAdapter.LegacyProviderAdapter(new FakeLegacyClient())
                        .complete(requestFrom(scenario)));
        assertExpectedError(scenario, error);
    }

    @Test
    void normalizesProviderError() {
        JsonNode scenario = contractCase("normalizes_provider_error");
        JsonNode providerError = scenario.path("input").path("provider_error");
        FakeLegacyClient client = new FakeLegacyClient();
        client.error = new ChatAdapter.ProviderException(
                providerError.path("code").asText(),
                providerError.path("message").asText());

        ChatAdapter.NormalizedException error = assertThrows(
                ChatAdapter.NormalizedException.class,
                () -> new ChatAdapter.LegacyProviderAdapter(client)
                        .complete(requestFrom(scenario)));
        assertExpectedError(scenario, error);
    }

    @Test
    void rejectsUnsupportedBeforeProviderCall() {
        JsonNode scenario = contractCase("rejects_unsupported_before_provider_call");
        FakeLegacyClient client = new FakeLegacyClient();

        ChatAdapter.NormalizedException error = assertThrows(
                ChatAdapter.NormalizedException.class,
                () -> new ChatAdapter.LegacyProviderAdapter(client)
                        .complete(requestFrom(scenario)));

        assertExpectedError(scenario, error);
        assertEquals(scenario.path("expected").path("provider_calls").asInt(), client.calls);
    }

    private static ChatAdapter.ChatRequest requestFrom(JsonNode scenario) {
        JsonNode request = scenario.path("input").path("request");
        List<ChatAdapter.Message> messages = new ArrayList<>();
        for (JsonNode message : request.path("messages")) {
            messages.add(new ChatAdapter.Message(
                    message.path("role").asText(),
                    message.path("content").asText()));
        }
        return new ChatAdapter.ChatRequest(
                request.path("model").asText(),
                messages,
                request.path("requires_tools").asBoolean());
    }

    private static void assertExpectedError(
            JsonNode scenario, ChatAdapter.NormalizedException error) {
        JsonNode expected = scenario.path("expected_error");
        assertEquals(expected.path("code").asText(), error.code());
        assertEquals(expected.path("retryable").asBoolean(), error.retryable());
    }

    private static JsonNode contractCase(String id) {
        for (JsonNode scenario : CONTRACT.path("cases")) {
            if (id.equals(scenario.path("id").asText())) {
                return scenario;
            }
        }
        throw new IllegalArgumentException("missing contract case: " + id);
    }

    private static JsonNode readContract() {
        try {
            return new ObjectMapper().readTree(Path.of(
                    "01-design-patterns/02-structural/01-adapter/fixtures/contract.json").toFile());
        } catch (IOException error) {
            throw new RuntimeException(error);
        }
    }
}
