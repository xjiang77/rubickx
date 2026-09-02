# spec — 04 Optimizer

> 子契约。权威范围见 vault [[SPEC - nanochat Foundations Learning Plan]]。本文件只定义本系统的 IO 契约 + 验收。

## IO 契约
- **输入 → 输出**:grads -> 参数更新

## 验收标准(Acceptance)
- [ ] `impl.py` 实现核心逻辑,`test_impl.py` 覆盖核心路径(目标单测 80%+)
- [ ] `parity.py`:同输入下与参考实现(优化器 (Muon / AdamW))输出/数值对齐(容差待定)
- [ ] `notes.md` 记录走读结论 + 实现取舍

## 对应需求
- SPEC: **R7**(Phase 1)

## 范围外(地基阶段不做)
- 极致性能 / kernel 优化(留 Phase 3)
