import org.junit.jupiter.api.Test;

        class TransactionalOutboxPatternTest {
            @Test void sharedContract() {
                ContractHarness.run("03-data-messaging-patterns/01-transactional-outbox/fixtures/contract.json", TransactionalOutboxPattern::evaluate);
            }
        }
