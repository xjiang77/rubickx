import org.junit.jupiter.api.Test;
class VisitorPatternTest{@Test void sharedContract(){ContractHarness.run("01-design-patterns/03-behavioral/10-visitor/fixtures/contract.json",VisitorPattern::evaluate);}}
