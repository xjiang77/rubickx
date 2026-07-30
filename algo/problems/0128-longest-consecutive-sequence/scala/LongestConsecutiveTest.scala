//> using scala 3.4.2
//> using test.dep org.scalameta::munit::1.0.0

class LongestConsecutiveTest extends munit.FunSuite:
  test("basic")           { assertEquals(LongestConsecutive.longestConsecutive(Array(100, 4, 200, 1, 3, 2)), 4) }
  test("with duplicates") { assertEquals(LongestConsecutive.longestConsecutive(Array(0, 3, 7, 2, 5, 8, 4, 6, 0, 1)), 9) }
  test("empty")           { assertEquals(LongestConsecutive.longestConsecutive(Array.empty[Int]), 0) }
  test("single")          { assertEquals(LongestConsecutive.longestConsecutive(Array(5)), 1) }
