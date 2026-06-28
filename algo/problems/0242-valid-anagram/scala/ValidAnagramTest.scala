//> using scala 3.4.2
//> using test.dep org.scalameta::munit::1.0.0

class ValidAnagramTest extends munit.FunSuite:
  test("valid")    { assert(ValidAnagram.isAnagram("anagram", "nagaram")) }
  test("invalid")  { assert(!ValidAnagram.isAnagram("rat", "car")) }
  test("diff len") { assert(!ValidAnagram.isAnagram("a", "ab")) }
