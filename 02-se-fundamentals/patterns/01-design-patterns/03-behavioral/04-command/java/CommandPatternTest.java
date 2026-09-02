import org.junit.jupiter.api.Test;
class CommandPatternTest{@Test void sharedContract(){ContractHarness.run("01-design-patterns/03-behavioral/04-command/fixtures/contract.json",CommandPattern::evaluate);}}
