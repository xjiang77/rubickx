import org.junit.jupiter.api.Test;

class SingletonPatternTest {
    @Test
    void sharedContract() {
        ContractHarness.run(
                "01-design-patterns/01-creational/05-singleton/fixtures/contract.json",
                SingletonPattern::evaluate);
    }
}
