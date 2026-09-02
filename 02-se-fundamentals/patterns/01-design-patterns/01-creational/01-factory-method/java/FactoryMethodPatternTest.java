import org.junit.jupiter.api.Test;

class FactoryMethodPatternTest {
    @Test
    void sharedContract() {
        ContractHarness.run(
                "01-design-patterns/01-creational/01-factory-method/fixtures/contract.json",
                FactoryMethodPattern::evaluate);
    }
}
