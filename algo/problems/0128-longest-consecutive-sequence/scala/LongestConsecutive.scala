object LongestConsecutive:
  /** 不可变 Set 去重;只取序列起点(x-1 不在集合),
    * Iterator.from(x) 惰性生成 x, x+1, ... 用 takeWhile(set) 截断得到长度。
    * maxOption 优雅处理空输入。均摊 O(n)/O(n)。 */
  def longestConsecutive(nums: Array[Int]): Int =
    val set = nums.toSet
    set.iterator
      .filterNot(x => set(x - 1))
      .map(x => Iterator.from(x).takeWhile(set).length)
      .maxOption
      .getOrElse(0)
