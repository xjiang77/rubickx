import org.junit.jupiter.api.Test;
class BulkheadPatternTest{@Test void sharedContract(){ContractHarness.run("02-reliability-patterns/04-bulkhead/fixtures/contract.json",BulkheadPattern::evaluate);}}
