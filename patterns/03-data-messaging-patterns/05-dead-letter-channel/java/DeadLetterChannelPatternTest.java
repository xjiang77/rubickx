import org.junit.jupiter.api.Test;

class DeadLetterChannelPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("03-data-messaging-patterns/05-dead-letter-channel/fixtures/contract.json", DeadLetterChannelPattern::evaluate);
    }
}
