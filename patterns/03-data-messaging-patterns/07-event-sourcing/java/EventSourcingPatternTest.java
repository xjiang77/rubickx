import org.junit.jupiter.api.Test;

class EventSourcingPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("03-data-messaging-patterns/07-event-sourcing/fixtures/contract.json", EventSourcingPattern::evaluate);
    }
}
