import org.junit.jupiter.api.Test;
class LoadSheddingPatternTest{@Test void sharedContract(){ContractHarness.run("02-reliability-patterns/05-load-shedding/fixtures/contract.json",LoadSheddingPattern::evaluate);}}
