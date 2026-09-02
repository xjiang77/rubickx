import org.junit.jupiter.api.Test;

class PrototypePatternTest {
    @Test
    void sharedContract() {
        ContractHarness.run(
                "01-design-patterns/01-creational/04-prototype/fixtures/contract.json",
                PrototypePattern::evaluate);
    }
}
