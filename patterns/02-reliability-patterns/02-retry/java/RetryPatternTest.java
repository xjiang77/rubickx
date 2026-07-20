import org.junit.jupiter.api.Test;
class RetryPatternTest{@Test void sharedContract(){ContractHarness.run("02-reliability-patterns/02-retry/fixtures/contract.json",RetryPattern::evaluate);}}
