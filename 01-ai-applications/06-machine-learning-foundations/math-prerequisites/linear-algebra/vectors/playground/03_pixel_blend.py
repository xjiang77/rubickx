#!/usr/bin/env python3
"""像素字母混合：把 5×5 的字母当作 25 维向量，用线性组合 (1−s)·A + s·B 做渐变。

运行：python3 03_pixel_blend.py
向量不一定是箭头：一张小图也是"有序的数字列表"（计算机科学视角），
加法和数乘对它同样成立，所以两张图之间可以取线性组合。
"""
A = ["  #  ",
     " # # ",
     "#####",
     "#   #",
     "#   #"]
B = ["#### ",
     "#   #",
     "#### ",
     "#   #",
     "#### "]
SHADES = " ░▒▓█"          # 0 … 1 的五档灰度


def to_vec(rows):
    return [1.0 if ch == "#" else 0.0 for row in rows for ch in row]


def blend(u, v, s):
    """(1−s)·u + s·v，逐分量计算。"""
    return [(1 - s) * x + s * y for x, y in zip(u, v)]


def render(vec, w=5):
    out = []
    for r in range(len(vec) // w):
        out.append("".join(SHADES[min(4, int(round(x * 4)))] for x in vec[r * w:(r + 1) * w]))
    return out


a, b = to_vec(A), to_vec(B)
steps = [0.0, 0.25, 0.5, 0.75, 1.0]
frames = [render(blend(a, b, s)) for s in steps]
print("      " + "      ".join(f"s={s:<4}" for s in steps))
for r in range(5):
    print("      " + "      ".join(f[r] + " " for f in frames))
print("\n每一列是 (1−s)·A + s·B 在 s 取 0、0.25、0.5、0.75、1 时的样子；中间的字母既不是 A 也不是 B，")
print("但它落在 A 与 B 的张成（一张二维平面）内。25 维空间里绝大多数点都不在这张平面上。")
print("\n试试：把 s 改成 −0.5 或 1.5（超出 0…1），线性组合仍然成立，只是像素值会越界，被截到 0…1 显示。")
