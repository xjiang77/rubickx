"""终端画板：把二维向量画在字符网格上。供 playground 各程序共用，只用标准库。

坐标约定：原点在网格中央，x 向右、y 向上；一个字符格代表 1 个单位。
"""

W, H = 41, 21  # 宽（x 从 −20 到 20）、高（y 从 −10 到 10）


class Canvas:
    def __init__(self, w=W, h=H):
        self.w, self.h = w, h
        self.cx, self.cy = w // 2, h // 2
        self.grid = [["·" if (x - self.cx) % 5 == 0 and (y - self.cy) % 5 == 0 else " " for x in range(w)] for y in range(h)]
        for x in range(w):
            self.grid[self.cy][x] = "─"
        for y in range(h):
            self.grid[y][self.cx] = "│"
        self.grid[self.cy][self.cx] = "┼"

    def put(self, x, y, ch):
        col, row = self.cx + round(x), self.cy - round(y)
        if 0 <= col < self.w and 0 <= row < self.h:
            self.grid[row][col] = ch

    def segment(self, p, q, ch):
        """从 p 到 q 画线段，端点不画（留给箭头标记）。"""
        (x0, y0), (x1, y1) = p, q
        n = max(1, int(2 * max(abs(x1 - x0), abs(y1 - y0))))
        for i in range(1, n):
            t = i / n
            self.put(x0 + t * (x1 - x0), y0 + t * (y1 - y0), ch)

    def arrow(self, v, ch="*", tail=(0, 0), label=None):
        """从 tail 出发画向量 v，尖端用 label（默认为 ch）标记。"""
        tip = (tail[0] + v[0], tail[1] + v[1])
        self.segment(tail, tip, ch)
        self.put(tip[0], tip[1], (label or ch)[0])
        if label and len(label) > 1:
            for k, c in enumerate(label[1:], 1):
                self.put(tip[0] + k, tip[1], c)
        return tip

    def render(self):
        return "\n".join("".join(row) for row in self.grid)


def show(canvas, title=None):
    if title:
        print(title)
    print(canvas.render())
