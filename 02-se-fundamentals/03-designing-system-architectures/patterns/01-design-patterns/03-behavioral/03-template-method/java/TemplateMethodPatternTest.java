import org.junit.jupiter.api.Test;
class TemplateMethodPatternTest{@Test void sharedContract(){ContractHarness.run("01-design-patterns/03-behavioral/03-template-method/fixtures/contract.json",TemplateMethodPattern::evaluate);}}
