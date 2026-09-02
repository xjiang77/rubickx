import org.junit.jupiter.api.Test;

class BuilderPatternTest {
    @Test
    void sharedContract() {
        ContractHarness.run(
                "01-design-patterns/01-creational/03-builder/fixtures/contract.json",
                BuilderPattern::evaluate);
    }
}
