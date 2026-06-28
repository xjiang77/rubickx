object GroupAnagrams:
  /** 用 groupBy(排序后的字符串) 一行聚合（Scala 最简写法）。O(n·k·log k)。 */
  def groupAnagrams(strs: Array[String]): List[List[String]] =
    strs.groupBy(s => s.sorted).values.map(_.toList).toList
