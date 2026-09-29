"""在 Jupyter 里画向量图：坐标网格、箭头、点、标签，输出内联 SVG。只用标准库。

用法：
    fig = Figure(xlim=(-1, 8), ylim=(-1, 5))
    fig.arrow((3, 1), label="v = (3, 1)")                 # 从原点出发
    fig.arrow((1, 2), tail=(6, 2), label="w", color=Figure.BLUE, dashed=True)
    fig.show()                                            # 在 notebook 里显示
颜色：INK（黑）为主向量，BLUE 为结果或强调，GREY 为辅助（首尾相接的中间段）。
"""

INK, BLUE, GREY, TINT = "#1f2523", "#2a63a5", "#66706c", "#eef3fa"


class Figure:
    INK, BLUE, GREY = INK, BLUE, GREY

    def __init__(self, xlim=(-1, 8), ylim=(-1, 5), unit=48, title=None):
        self.xlim, self.ylim, self.unit, self.title = xlim, ylim, unit, title
        self.w = (xlim[1] - xlim[0]) * unit + 40
        self.h = (ylim[1] - ylim[0]) * unit + (60 if title else 40)
        self.items = []
        self._marker_ids = {}

    def _xy(self, p):
        x = 20 + (p[0] - self.xlim[0]) * self.unit
        y = (self.h - 20) - (p[1] - self.ylim[0]) * self.unit
        return x, y

    def arrow(self, v, tail=(0, 0), label=None, color=INK, dashed=False, width=2.5, label_pos="tip", side=1):
        """从 tail 画到 tail + v，尖端带箭头。label_pos="tip" 把标签放在尖端旁，"mid" 放在线段中点的一侧（side=±1 选侧）。"""
        tip = (tail[0] + v[0], tail[1] + v[1])
        (x0, y0), (x1, y1) = self._xy(tail), self._xy(tip)
        mid = f'marker-end="url(#m{self._marker(color)})"'
        dash = ' stroke-dasharray="7 5"' if dashed else ""
        self.items.append(f'<line x1="{x0:.1f}" y1="{y0:.1f}" x2="{x1:.1f}" y2="{y1:.1f}" stroke="{color}" stroke-width="{width}"{dash} {mid}/>')
        if label:
            if label_pos == "mid":
                mx, my = (x0 + x1) / 2, (y0 + y1) / 2
                dx, dy = x1 - x0, y1 - y0
                n = (dx * dx + dy * dy) ** 0.5 or 1
                px, py = -dy / n * 20 * side, dx / n * 20 * side
                self.items.append(f'<text x="{mx + px:.1f}" y="{my + py + 5:.1f}" fill="{color}" font-size="15" text-anchor="middle">{label}</text>')
            else:
                dx = 10 if v[0] >= 0 else -10
                anchor = "start" if v[0] >= 0 else "end"
                self.items.append(f'<text x="{x1 + dx:.1f}" y="{y1 - 8:.1f}" fill="{color}" font-size="15" text-anchor="{anchor}">{label}</text>')
        return tip

    def point(self, p, label=None, color=INK, r=4):
        x, y = self._xy(p)
        self.items.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="{r}" fill="{color}"/>')
        if label:
            self.items.append(f'<text x="{x + 8:.1f}" y="{y - 6:.1f}" fill="{color}" font-size="15">{label}</text>')

    def polygon(self, pts, fill=TINT, stroke=GREY):
        s = " ".join(f"{x:.1f},{y:.1f}" for x, y in (self._xy(p) for p in pts))
        self.items.insert(0, f'<polygon points="{s}" fill="{fill}" stroke="{stroke}" stroke-width="1.5"/>')

    def text(self, p, s, color=GREY, size=14):
        x, y = self._xy(p)
        self.items.append(f'<text x="{x:.1f}" y="{y:.1f}" fill="{color}" font-size="{size}">{s}</text>')

    def _marker(self, color):
        if color not in self._marker_ids:
            self._marker_ids[color] = len(self._marker_ids)
        return self._marker_ids[color]

    def svg(self):
        grid = []
        for gx in range(self.xlim[0], self.xlim[1] + 1):
            x, _ = self._xy((gx, 0))
            grid.append(f'<line x1="{x:.1f}" y1="{self._xy((0, self.ylim[1]))[1]:.1f}" x2="{x:.1f}" y2="{self._xy((0, self.ylim[0]))[1]:.1f}" stroke="#e3e7e5" stroke-width="1"/>')
        for gy in range(self.ylim[0], self.ylim[1] + 1):
            _, y = self._xy((0, gy))
            grid.append(f'<line x1="{self._xy((self.xlim[0], 0))[0]:.1f}" y1="{y:.1f}" x2="{self._xy((self.xlim[1], 0))[0]:.1f}" y2="{y:.1f}" stroke="#e3e7e5" stroke-width="1"/>')
        ox, oy = self._xy((0, 0))
        axes = (f'<line x1="{self._xy((self.xlim[0], 0))[0]:.1f}" y1="{oy:.1f}" x2="{self._xy((self.xlim[1], 0))[0]:.1f}" y2="{oy:.1f}" stroke="{GREY}" stroke-width="1.5"/>'
                f'<line x1="{ox:.1f}" y1="{self._xy((0, self.ylim[1]))[1]:.1f}" x2="{ox:.1f}" y2="{self._xy((0, self.ylim[0]))[1]:.1f}" stroke="{GREY}" stroke-width="1.5"/>')
        ticks = "".join(f'<text x="{self._xy((gx, 0))[0]:.1f}" y="{oy + 16:.1f}" fill="{GREY}" font-size="11" text-anchor="middle">{gx}</text>'
                        for gx in range(self.xlim[0], self.xlim[1] + 1) if gx)
        ticks += "".join(f'<text x="{ox - 6:.1f}" y="{self._xy((0, gy))[1] + 4:.1f}" fill="{GREY}" font-size="11" text-anchor="end">{gy}</text>'
                         for gy in range(self.ylim[0], self.ylim[1] + 1) if gy)
        markers = "".join(f'<marker id="m{i}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="12" markerHeight="12" markerUnits="userSpaceOnUse" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="{c}"/></marker>'
                          for c, i in self._marker_ids.items())
        title = f'<text x="20" y="22" fill="{INK}" font-size="16" font-weight="600">{self.title}</text>' if self.title else ""
        body = "".join(self.items)
        return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {self.w} {self.h}" width="{self.w}" height="{self.h}" '
                f'style="font-family:-apple-system,Segoe UI,PingFang SC,sans-serif;background:#fff"><defs>{markers}</defs>'
                f'{title}{"".join(grid)}{axes}{ticks}{body}</svg>')

    def show(self):
        from IPython.display import SVG, display
        display(SVG(self.svg()))

    def _repr_svg_(self):
        return self.svg()
