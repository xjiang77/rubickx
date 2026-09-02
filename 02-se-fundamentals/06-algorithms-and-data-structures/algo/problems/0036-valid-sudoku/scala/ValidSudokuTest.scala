//> using scala 3.4.2
//> using test.dep org.scalameta::munit::1.0.0

class ValidSudokuTest extends munit.FunSuite:
  private val valid = Vector(
    "53..7....",
    "6..195...",
    ".98....6.",
    "8...6...3",
    "4..8.3..1",
    "7...2...6",
    ".6....28.",
    "...419..5",
    "....8..79",
  )

  private def toBoard(rows: Vector[String]): Array[Array[Char]] =
    rows.map(_.toCharArray).toArray

  test("valid board") { assert(ValidSudoku.isValidSudoku(toBoard(valid))) }

  test("box duplicate") {
    val rows = valid.updated(0, "8" + valid(0).drop(1)) // (0,0) 5→8,与 (2,2) 的 8 同 box
    assert(!ValidSudoku.isValidSudoku(toBoard(rows)))
  }

  test("row duplicate") {
    val rows = valid.updated(0, valid(0).take(2) + "5" + valid(0).drop(3)) // 第 0 行两个 5
    assert(!ValidSudoku.isValidSudoku(toBoard(rows)))
  }
