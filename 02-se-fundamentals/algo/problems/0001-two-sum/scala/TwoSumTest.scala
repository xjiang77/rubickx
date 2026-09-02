//> using scala 3.4.2
//> using test.dep org.scalameta::munit::1.0.0

class TwoSumTest extends munit.FunSuite:
  test("basic")  { assertEquals(TwoSum.twoSum(Array(2, 7, 11, 15), 9).toList, List(0, 1)) }
  test("middle") { assertEquals(TwoSum.twoSum(Array(3, 2, 4), 6).toList, List(1, 2)) }
  test("same")   { assertEquals(TwoSum.twoSum(Array(3, 3), 6).toList, List(0, 1)) }
