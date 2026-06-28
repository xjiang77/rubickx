object ContainsDuplicate:
  /** mutable.HashSet.add 返回 false 即已存在。O(n)/O(n)。
    * 函数式一行：nums.toSet.size != nums.length（牺牲提前退出）。 */
  def containsDuplicate(nums: Array[Int]): Boolean =
    val seen = scala.collection.mutable.HashSet.empty[Int]
    var i = 0
    var found = false
    while i < nums.length && !found do
      if !seen.add(nums(i)) then found = true
      i += 1
    found
