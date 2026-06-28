# 练习 — 从零 micrograd (D2 / R3)

> C5 硬约束:**本人手写**,agent 不代写。卡住可让 agent 讲概念 / review,但不要成品。

## 目标
手写一个最小自动微分引擎(标量 `Value`,支持 `+ * tanh/relu`,反向传播),
搭一个 MLP,在小任务上收敛。

## 验收(R3)
- [ ] `Value` 支持前向 + `.backward()` 反向
- [ ] MLP 在小任务(如玩具二分类)上 loss 下降、收敛
- [ ] **梯度与数值梯度对齐**(有限差分校验,误差 < 1e-4)
- [ ] 复盘写入 vault [[nanochat - Experiment Log]]

## 走读连接
对照 nanochat_mlx/train.py 的 `fwd -> loss -> backward -> step` 主链,理解真实框架里这条链长什么样。
