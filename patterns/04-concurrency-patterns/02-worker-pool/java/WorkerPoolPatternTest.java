import org.junit.jupiter.api.Test;

class WorkerPoolPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("04-concurrency-patterns/02-worker-pool/fixtures/contract.json", WorkerPoolPattern::evaluate);
    }
}
