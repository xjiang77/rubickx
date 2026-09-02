import org.junit.jupiter.api.Test;
class CircuitBreakerPatternTest{@Test void sharedContract(){ContractHarness.run("02-reliability-patterns/03-circuit-breaker/fixtures/contract.json",CircuitBreakerPattern::evaluate);}}
