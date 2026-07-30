object ValidSudoku:
  /** for 推导式把每个已填格子展开成 (行,值)/(列,值)/(box,值) 三条"证据",
    * distinct 后长度不变 ⇔ 无重复。声明式一遍写完;命令式 bitmask 版见 NOTES。 */
  def isValidSudoku(board: Array[Array[Char]]): Boolean =
    val seen = for
      r <- 0 until 9
      c <- 0 until 9
      v = board(r)(c)
      if v != '.'
      key <- Seq(("r", r, v), ("c", c, v), ("b", (r / 3) * 3 + c / 3, v))
    yield key
    seen.distinct.size == seen.size
