import org.junit.jupiter.api.Test;

class AbstractFactoryPatternTest {
    @Test
    void sharedContract() {
        ContractHarness.run(
                "01-design-patterns/01-creational/02-abstract-factory/fixtures/contract.json",
                AbstractFactoryPattern::evaluate);
    }
}
