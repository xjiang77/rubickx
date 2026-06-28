object TwoSum:
  /** 一次遍历 + mutable.Map：seen(值)=下标，命中 target-x 即返回。O(n)/O(n)。
    * 用 while + 标志位避免 `return`（Scala 习惯尽量不用 return）。 */
  def twoSum(nums: Array[Int], target: Int): Array[Int] =
    val seen = scala.collection.mutable.HashMap.empty[Int, Int]
    var res = Array.empty[Int]
    var i = 0
    while i < nums.length && res.isEmpty do
      val need = target - nums(i)
      seen.get(need) match
        case Some(j) => res = Array(j, i)
        case None    => seen(nums(i)) = i
      i += 1
    res
