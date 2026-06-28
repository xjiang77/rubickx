//> using scala 3.4.2
//> using test.dep org.scalameta::munit::1.0.0

class GroupAnagramsTest extends munit.FunSuite:
  private def norm(gs: List[List[String]]): List[List[String]] =
    gs.map(_.sorted).sortBy(_.mkString(","))

  test("basic"):
    val res = GroupAnagrams.groupAnagrams(Array("eat", "tea", "tan", "ate", "nat", "bat"))
    val want = List(List("bat"), List("nat", "tan"), List("ate", "eat", "tea"))
    assertEquals(norm(res), norm(want))

  test("empty string"):
    assertEquals(norm(GroupAnagrams.groupAnagrams(Array(""))), List(List("")))
