import org.junit.jupiter.api.Test;
class TimeoutPatternTest{@Test void sharedContract(){ContractHarness.run("02-reliability-patterns/01-timeout/fixtures/contract.json",TimeoutPattern::evaluate);}}
