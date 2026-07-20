import org.junit.jupiter.api.Test;

class PublisherSubscriberPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("03-data-messaging-patterns/04-publisher-subscriber/fixtures/contract.json", PublisherSubscriberPattern::evaluate);
    }
}
