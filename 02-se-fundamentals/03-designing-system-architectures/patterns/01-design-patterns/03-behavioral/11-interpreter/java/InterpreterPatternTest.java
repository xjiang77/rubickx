import org.junit.jupiter.api.Test;
class InterpreterPatternTest{@Test void sharedContract(){ContractHarness.run("01-design-patterns/03-behavioral/11-interpreter/fixtures/contract.json",InterpreterPattern::evaluate);}}
