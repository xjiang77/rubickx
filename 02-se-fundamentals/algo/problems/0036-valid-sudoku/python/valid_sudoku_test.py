import os, sys
sys.path.insert(0, os.path.dirname(__file__))
from valid_sudoku import is_valid_sudoku

VALID = [
    "53..7....",
    "6..195...",
    ".98....6.",
    "8...6...3",
    "4..8.3..1",
    "7...2...6",
    ".6....28.",
    "...419..5",
    "....8..79",
]


def to_board(rows: list[str]) -> list[list[str]]:
    return [list(row) for row in rows]


def test_valid():
    assert is_valid_sudoku(to_board(VALID)) is True

def test_box_duplicate():
    rows = ["8" + VALID[0][1:]] + VALID[1:]  # (0,0) 5→8,与 (2,2) 的 8 同 box
    assert is_valid_sudoku(to_board(rows)) is False

def test_row_duplicate():
    rows = [VALID[0][:2] + "5" + VALID[0][3:]] + VALID[1:]  # 第 0 行出现两个 5
    assert is_valid_sudoku(to_board(rows)) is False
