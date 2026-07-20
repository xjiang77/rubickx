import org.junit.jupiter.api.Test;

class SagaPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("03-data-messaging-patterns/03-saga/fixtures/contract.json", SagaPattern::evaluate);
    }
}
