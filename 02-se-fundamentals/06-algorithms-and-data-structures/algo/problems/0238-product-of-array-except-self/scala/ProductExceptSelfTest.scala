//> using scala 3.4.2
//> using test.dep org.scalameta::munit::1.0.0

class ProductExceptSelfTest extends munit.FunSuite:
  test("basic")        { assertEquals(ProductExceptSelf.productExceptSelf(Array(1, 2, 3, 4)).toList, List(24, 12, 8, 6)) }
  test("with zero")    { assertEquals(ProductExceptSelf.productExceptSelf(Array(-1, 1, 0, -3, 3)).toList, List(0, 0, 9, 0, 0)) }
  test("two elements") { assertEquals(ProductExceptSelf.productExceptSelf(Array(2, 3)).toList, List(3, 2)) }
