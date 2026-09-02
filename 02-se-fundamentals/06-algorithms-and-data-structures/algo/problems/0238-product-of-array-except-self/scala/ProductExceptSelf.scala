object ProductExceptSelf:
  /** scanLeft/scanRight 生成前缀积与后缀积(各 n+1 长),res(i) = prefix(i) * suffix(i+1)。
    * 声明式写法,额外空间 O(n);O(1) 额外空间的命令式版见 NOTES。 */
  def productExceptSelf(nums: Array[Int]): Array[Int] =
    val prefix = nums.scanLeft(1)(_ * _)   // prefix(i) = nums(0..i-1) 之积
    val suffix = nums.scanRight(1)(_ * _)  // suffix(i) = nums(i..n-1) 之积
    Array.tabulate(nums.length)(i => prefix(i) * suffix(i + 1))
