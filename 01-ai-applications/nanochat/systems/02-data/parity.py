"""02 Data — 与参考实现对拍(parity)。

骨架可由 agent 协助;核心 impl 由本人手写。
用法:同一输入分别喂给 你的 impl 和参考(数据加载 / 分片 / batching),比较输出/数值。
"""
# from impl import ...
# import sys; sys.path.insert(0, "../../deps/nanochat-mlx")  # 参考实现

TOL = 1e-4  # 数值对齐容差(按系统调整)


def run_parity():
    raise NotImplementedError("TODO: 搭好 你的impl vs 参考 的同输入对拍")


if __name__ == "__main__":
    run_parity()
