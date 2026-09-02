//> using scala 3.4.2
//> using test.dep org.scalameta::munit::1.0.0

class ContainsDuplicateTest extends munit.FunSuite:
  test("has dup") { assert(ContainsDuplicate.containsDuplicate(Array(1, 2, 3, 1))) }
  test("unique")  { assert(!ContainsDuplicate.containsDuplicate(Array(1, 2, 3, 4))) }
  test("empty")   { assert(!ContainsDuplicate.containsDuplicate(Array.empty[Int])) }
