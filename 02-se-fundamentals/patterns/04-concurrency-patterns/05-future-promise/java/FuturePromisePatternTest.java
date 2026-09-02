import org.junit.jupiter.api.Test;

class FuturePromisePatternTest {
    @Test void sharedContract() {
        ContractHarness.run("04-concurrency-patterns/05-future-promise/fixtures/contract.json", FuturePromisePattern::evaluate);
    }
}
