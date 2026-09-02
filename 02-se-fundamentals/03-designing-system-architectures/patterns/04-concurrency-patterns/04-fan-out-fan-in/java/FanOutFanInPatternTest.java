import org.junit.jupiter.api.Test;

class FanOutFanInPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("04-concurrency-patterns/04-fan-out-fan-in/fixtures/contract.json", FanOutFanInPattern::evaluate);
    }
}
