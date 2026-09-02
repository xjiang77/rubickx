import org.junit.jupiter.api.Test;

class IdempotentConsumerInboxPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("03-data-messaging-patterns/02-idempotent-consumer-inbox/fixtures/contract.json", IdempotentConsumerInboxPattern::evaluate);
    }
}
