import org.junit.jupiter.api.Test;
class HedgedRequestsPatternTest{@Test void sharedContract(){ContractHarness.run("02-reliability-patterns/06-hedged-requests/fixtures/contract.json",HedgedRequestsPattern::evaluate);}}
