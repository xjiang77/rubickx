import org.junit.jupiter.api.Test;

class BoundedProducerConsumerPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("04-concurrency-patterns/01-bounded-producer-consumer/fixtures/contract.json", BoundedProducerConsumerPattern::evaluate);
    }
}
