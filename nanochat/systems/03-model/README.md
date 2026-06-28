# 03 · Model (GPT)

> nanochat 系统重写 · 对应 SPEC 需求 **R5** · Phase 1

## 这是什么
把 nanochat 的「Model (GPT)」系统**自己从零重写一遍**,再与参考实现对拍。不照抄。

## 参考代码(只读对照)
- 参考实现:nanochat_mlx/gpt.py
- 位置:`../../deps/nanochat-mlx/`(submodule)

## 设计要点
- (走读后填:核心数据流、关键不变量、易错点)

## 五件套
`spec.md`(子契约) · `impl.py`(你的实现) · `test_impl.py`(单测) · `parity.py`(与参考对拍) · `notes.md`(走读+实现笔记)

## 完成判定
`test_impl.py` 通过 + `parity.py` 与参考数值对齐。durable insight 晋升 vault `03_Slipbox/01_AI/`。
