object ValidAnagram:
  /** 用 groupMapReduce 统计两边字符频次再比较（Scala 2.13+/3 的现代写法）。O(n)/O(k)。 */
  def isAnagram(s: String, t: String): Boolean =
    if s.length != t.length then false
    else
      def freq(str: String): Map[Char, Int] =
        str.groupMapReduce(identity)(_ => 1)(_ + _)
      freq(s) == freq(t)
