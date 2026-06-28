"""从零 micrograd —— 本人手写 (C5)。下面只有骨架,实现留空。"""


class Value:
    def __init__(self, data, _children=(), _op=""):
        self.data = data
        self.grad = 0.0
        # TODO: _backward, _prev, _op

    def __add__(self, other):
        raise NotImplementedError

    def __mul__(self, other):
        raise NotImplementedError

    def backward(self):
        raise NotImplementedError
