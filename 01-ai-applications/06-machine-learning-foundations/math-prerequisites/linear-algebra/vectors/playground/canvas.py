"""终端画板：把二维向量画在字符网格上。供 playground 各程序共用，只用标准库。

坐标约定：原点在网格中央，x 向右、y 向上；一个字符格代表 1 个单位。
只适合终端里的粗略示意；notebook 里用 svgplot.Figure 画精确的图。

Examples:
    >>> canvas = Canvas(w=9, h=5)
    >>> canvas.arrow((3, 1), "*", label="v")
    (3, 1)
    >>> print(canvas.render())
        │
        │ *v
    ────***──
        │
        │
"""
from typing import TypeAlias

Point: TypeAlias = tuple[float, float]

W, H = 41, 21
"""默认宽高：x 从 −20 到 20，y 从 −10 到 10。"""


class Canvas:
    """一张字符网格，带坐标轴与每 5 格一个的参考点。

    Attributes:
        w, h: 网格的列数与行数。
        cx, cy: 原点所在的列与行。
        grid: 二维字符列表，grid[row][col]。
    """

    def __init__(self, w: int = W, h: int = H) -> None:
        self.w, self.h = w, h
        self.cx, self.cy = w // 2, h // 2
        self.grid = [
            ["·" if (x - self.cx) % 5 == 0 and (y - self.cy) % 5 == 0 else " " for x in range(w)]
            for y in range(h)
        ]
        for x in range(w):
            self.grid[self.cy][x] = "─"
        for y in range(h):
            self.grid[y][self.cx] = "│"
        self.grid[self.cy][self.cx] = "┼"

    def put(self, x: float, y: float, ch: str) -> None:
        """在数学坐标 (x, y) 最近的格子写一个字符；越界的点忽略。"""
        col, row = self.cx + round(x), self.cy - round(y)
        if 0 <= col < self.w and 0 <= row < self.h:
            self.grid[row][col] = ch

    def segment(self, p: Point, q: Point, ch: str) -> None:
        """从 p 到 q 画线段，端点不画（留给箭头标记）。"""
        (x0, y0), (x1, y1) = p, q
        n = max(1, int(2 * max(abs(x1 - x0), abs(y1 - y0))))
        for i in range(1, n):
            t = i / n
            self.put(x0 + t * (x1 - x0), y0 + t * (y1 - y0), ch)

    def arrow(self, v: Point, ch: str = "*", tail: Point = (0, 0), label: str | None = None) -> Point:
        """从 tail 出发画向量 v，尖端用 label 的首字符标记，返回尖端坐标。

        Args:
            v: 位移向量。
            ch: 线段用的字符。
            tail: 起点；首尾相接时传上一段的尖端。
            label: 尖端标签；多于一个字符时向右依次写出。

        Returns:
            尖端的数学坐标。
        """
        tip = (tail[0] + v[0], tail[1] + v[1])
        self.segment(tail, tip, ch)
        self.put(tip[0], tip[1], (label or ch)[0])
        if label and len(label) > 1:
            for k, c in enumerate(label[1:], 1):
                self.put(tip[0] + k, tip[1], c)
        return tip

    def render(self) -> str:
        """把网格拼成多行字符串（行尾空格去掉）。"""
        return "\n".join("".join(row).rstrip() for row in self.grid)


def show(canvas: Canvas, title: str | None = None) -> None:
    """打印画板，可带标题行。"""
    if title:
        print(title)
    print(canvas.render())


if __name__ == "__main__":
    import doctest

    doctest.testmod(verbose=True)
