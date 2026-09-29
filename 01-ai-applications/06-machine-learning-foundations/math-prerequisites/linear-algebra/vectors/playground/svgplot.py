"""在 Jupyter 里画向量图：坐标网格、箭头、点、标签，输出内联 SVG。只用标准库。

Examples:
    >>> fig = Figure(xlim=(-1, 4), ylim=(-1, 3), title="v = (3, 1)")
    >>> tip = fig.arrow((3, 1), label="v")
    >>> tip
    (3, 1)
    >>> fig.svg().startswith("<svg")
    True

颜色约定：INK（黑）画主向量，BLUE 画结果或强调，GREY 画辅助段（首尾相接的中间段）。
在 notebook 里调用 ``fig.show()`` 显示；作为单元格最后一个表达式时也会自动显示。
"""

from typing import TypeAlias

Point: TypeAlias = tuple[float, float]
"""平面上的点或向量，(x, y)。"""

INK = "#1f2523"
BLUE = "#2a63a5"
GREY = "#66706c"
TINT = "#eef3fa"
_GRID = "#e3e7e5"


class Figure:
    """一张带坐标网格的向量图。

    坐标以数学约定为准：x 向右、y 向上、原点 (0, 0)；``unit`` 是 1 个单位对应的像素数。
    所有绘制方法都接受数学坐标，内部换算成 SVG 像素。

    Attributes:
        xlim: x 轴范围 (最小值, 最大值)，整数。
        ylim: y 轴范围 (最小值, 最大值)，整数。
        unit: 每单位像素数。
        title: 图标题，None 表示不画。
    """

    INK, BLUE, GREY = INK, BLUE, GREY

    def __init__(
        self,
        xlim: tuple[int, int] = (-1, 8),
        ylim: tuple[int, int] = (-1, 5),
        unit: int = 48,
        title: str | None = None,
    ) -> None:
        self.xlim, self.ylim, self.unit, self.title = xlim, ylim, unit, title
        self.width = (xlim[1] - xlim[0]) * unit + 40
        self.height = (ylim[1] - ylim[0]) * unit + (60 if title else 40)
        self._items: list[str] = []
        self._marker_ids: dict[str, int] = {}

    def _pixel(self, p: Point) -> Point:
        """把数学坐标换算成 SVG 像素坐标（y 轴翻转）。

        Examples:
            >>> Figure(xlim=(0, 2), ylim=(0, 2), unit=10)._pixel((0, 0))   # 高 60：网格 20 + 边距 40
            (20, 40)
        """
        x = 20 + (p[0] - self.xlim[0]) * self.unit
        y = (self.height - 20) - (p[1] - self.ylim[0]) * self.unit
        return x, y

    def arrow(
        self,
        v: Point,
        tail: Point = (0, 0),
        label: str | None = None,
        color: str = INK,
        dashed: bool = False,
        width: float = 2.5,
        label_pos: str = "tip",
        side: int = 1,
    ) -> Point:
        """画从 tail 出发、位移为 v 的箭头，返回尖端坐标。

        Args:
            v: 箭头的位移向量。
            tail: 起点；首尾相接时传上一段的尖端。
            label: 标签文字，None 不画。
            color: 线与标签的颜色。
            dashed: 是否虚线。
            width: 线宽（像素）。
            label_pos: "tip" 把标签放在尖端旁；"mid" 放在线段中点的一侧。
            side: label_pos 为 "mid" 时选哪一侧，1 或 −1。

        Returns:
            尖端的数学坐标 tail + v，可作为下一段的 tail。

        Examples:
            >>> fig = Figure()
            >>> fig.arrow((3, 1))
            (3, 1)
            >>> fig.arrow((1, 2), tail=(3, 1))
            (4, 3)
        """
        tip = (tail[0] + v[0], tail[1] + v[1])
        (x0, y0), (x1, y1) = self._pixel(tail), self._pixel(tip)
        marker = f'marker-end="url(#m{self._marker(color)})"'
        dash = ' stroke-dasharray="7 5"' if dashed else ""
        self._items.append(
            f'<line x1="{x0:.1f}" y1="{y0:.1f}" x2="{x1:.1f}" y2="{y1:.1f}" '
            f'stroke="{color}" stroke-width="{width}"{dash} {marker}/>'
        )
        if label:
            self._items.append(self._arrow_label(label, color, (x0, y0), (x1, y1), v, label_pos, side))
        return tip

    @staticmethod
    def _arrow_label(
        label: str, color: str, start: Point, end: Point, v: Point, label_pos: str, side: int
    ) -> str:
        """生成箭头标签的 <text> 元素；"mid" 模式把标签推离线段 20 像素，避免压线。"""
        (x0, y0), (x1, y1) = start, end
        if label_pos == "mid":
            mx, my = (x0 + x1) / 2, (y0 + y1) / 2
            dx, dy = x1 - x0, y1 - y0
            length = (dx * dx + dy * dy) ** 0.5 or 1
            px, py = -dy / length * 20 * side, dx / length * 20 * side
            return (
                f'<text x="{mx + px:.1f}" y="{my + py + 5:.1f}" fill="{color}" '
                f'font-size="15" text-anchor="middle">{label}</text>'
            )
        offset = 10 if v[0] >= 0 else -10
        anchor = "start" if v[0] >= 0 else "end"
        return (
            f'<text x="{x1 + offset:.1f}" y="{y1 - 8:.1f}" fill="{color}" '
            f'font-size="15" text-anchor="{anchor}">{label}</text>'
        )

    def point(self, p: Point, label: str | None = None, color: str = INK, r: float = 4) -> None:
        """在 p 处画一个实心圆点，可带标签。"""
        x, y = self._pixel(p)
        self._items.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="{r}" fill="{color}"/>')
        if label:
            self._items.append(
                f'<text x="{x + 8:.1f}" y="{y - 6:.1f}" fill="{color}" font-size="15">{label}</text>'
            )

    def polygon(self, points: list[Point], fill: str = TINT, stroke: str = GREY) -> None:
        """画一个填充多边形，放在所有已画元素之下（用作背景区域）。"""
        coords = " ".join(f"{x:.1f},{y:.1f}" for x, y in (self._pixel(p) for p in points))
        self._items.insert(
            0, f'<polygon points="{coords}" fill="{fill}" stroke="{stroke}" stroke-width="1.5"/>'
        )

    def text(self, p: Point, s: str, color: str = GREY, size: int = 14) -> None:
        """在 p 处写一段文字。"""
        x, y = self._pixel(p)
        self._items.append(f'<text x="{x:.1f}" y="{y:.1f}" fill="{color}" font-size="{size}">{s}</text>')

    def _marker(self, color: str) -> int:
        """为每种颜色登记一个箭头 marker，返回其编号。"""
        if color not in self._marker_ids:
            self._marker_ids[color] = len(self._marker_ids)
        return self._marker_ids[color]

    def _grid_and_axes(self) -> str:
        """网格线、坐标轴与刻度数字。"""
        x_left, _ = self._pixel((self.xlim[0], 0))
        x_right, _ = self._pixel((self.xlim[1], 0))
        _, y_top = self._pixel((0, self.ylim[1]))
        _, y_bottom = self._pixel((0, self.ylim[0]))
        ox, oy = self._pixel((0, 0))
        parts = []
        for gx in range(self.xlim[0], self.xlim[1] + 1):
            x, _ = self._pixel((gx, 0))
            parts.append(f'<line x1="{x:.1f}" y1="{y_top:.1f}" x2="{x:.1f}" y2="{y_bottom:.1f}" stroke="{_GRID}"/>')
            if gx:
                parts.append(f'<text x="{x:.1f}" y="{oy + 16:.1f}" fill="{GREY}" font-size="11" text-anchor="middle">{gx}</text>')
        for gy in range(self.ylim[0], self.ylim[1] + 1):
            _, y = self._pixel((0, gy))
            parts.append(f'<line x1="{x_left:.1f}" y1="{y:.1f}" x2="{x_right:.1f}" y2="{y:.1f}" stroke="{_GRID}"/>')
            if gy:
                parts.append(f'<text x="{ox - 6:.1f}" y="{y + 4:.1f}" fill="{GREY}" font-size="11" text-anchor="end">{gy}</text>')
        parts.append(f'<line x1="{x_left:.1f}" y1="{oy:.1f}" x2="{x_right:.1f}" y2="{oy:.1f}" stroke="{GREY}" stroke-width="1.5"/>')
        parts.append(f'<line x1="{ox:.1f}" y1="{y_top:.1f}" x2="{ox:.1f}" y2="{y_bottom:.1f}" stroke="{GREY}" stroke-width="1.5"/>')
        return "".join(parts)

    def svg(self) -> str:
        """返回完整的 SVG 字符串。"""
        markers = "".join(
            f'<marker id="m{i}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="12" markerHeight="12" '
            f'markerUnits="userSpaceOnUse" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="{c}"/></marker>'
            for c, i in self._marker_ids.items()
        )
        title = f'<text x="20" y="22" fill="{INK}" font-size="16" font-weight="600">{self.title}</text>' if self.title else ""
        return (
            f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {self.width} {self.height}" '
            f'width="{self.width}" height="{self.height}" '
            f'style="font-family:-apple-system,Segoe UI,PingFang SC,sans-serif;background:#fff">'
            f"<defs>{markers}</defs>{title}{self._grid_and_axes()}{''.join(self._items)}</svg>"
        )

    def show(self) -> None:
        """在 notebook 里显示（延迟导入 IPython，使本模块在没有 IPython 的环境也能导入）。"""
        from IPython.display import SVG, display

        display(SVG(self.svg()))

    def _repr_svg_(self) -> str:
        return self.svg()


if __name__ == "__main__":
    import doctest

    doctest.testmod(verbose=True)
