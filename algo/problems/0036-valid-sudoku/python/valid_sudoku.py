def is_valid_sudoku(board: list[list[str]]) -> bool:
    """rows/cols/boxes 三组 set 记录已见数字,box 下标 = (r//3)*3 + c//3。
    一次扫描,任何一处重复即 False。时间 O(81),空间 O(81)。"""
    rows: list[set[str]] = [set() for _ in range(9)]
    cols: list[set[str]] = [set() for _ in range(9)]
    boxes: list[set[str]] = [set() for _ in range(9)]
    for r in range(9):
        for c in range(9):
            v = board[r][c]
            if v == ".":
                continue
            b = (r // 3) * 3 + c // 3
            if v in rows[r] or v in cols[c] or v in boxes[b]:
                return False
            rows[r].add(v)
            cols[c].add(v)
            boxes[b].add(v)
    return True
