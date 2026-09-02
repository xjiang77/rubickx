object TopKFrequent:
  /** groupMapReduce 一行计数(2.13+ 标准库),按频次降序取前 k。
    * 排序版 O(n log n),胜在表达力;O(n) 桶排序版见 NOTES。 */
  def topKFrequent(nums: Array[Int], k: Int): Array[Int] =
    nums
      .groupMapReduce(identity)(_ => 1)(_ + _)
      .toArray
      .sortBy(-_._2)
      .take(k)
      .map(_._1)
