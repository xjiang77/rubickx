//> using scala 3.4.2
//> using test.dep org.scalameta::munit::1.0.0

class TopKFrequentTest extends munit.FunSuite:
  test("basic")     { assertEquals(TopKFrequent.topKFrequent(Array(1, 1, 1, 2, 2, 3), 2).sorted.toList, List(1, 2)) }
  test("single")    { assertEquals(TopKFrequent.topKFrequent(Array(1), 1).toList, List(1)) }
  test("same freq") { assertEquals(TopKFrequent.topKFrequent(Array(4, 5, 6), 3).sorted.toList, List(4, 5, 6)) }
