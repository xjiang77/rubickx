import org.junit.jupiter.api.Test;

class PipelinePatternTest {
    @Test void sharedContract() {
        ContractHarness.run("04-concurrency-patterns/03-pipeline/fixtures/contract.json", PipelinePattern::evaluate);
    }
}
