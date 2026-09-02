import org.junit.jupiter.api.Test;

class CqrsPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("03-data-messaging-patterns/06-cqrs/fixtures/contract.json", CqrsPattern::evaluate);
    }
}
