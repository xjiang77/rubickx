# rubickx — agent 规范

## 练习骨架规范

适用于 C5 练习（本人手写）的骨架文件，例如 `math-prerequisites/linear-algebra/<目录>/<name>.py`。样例：`linear-algebra/vectors/vectors.py`。

- 函数体只写 `raise NotImplementedError`，不给实现（C5）。学习者明确要求时，实现只在对话里给出，由本人输入，不写进文件。
- 类型注解：模块顶部定义类型别名（如 `Vector: TypeAlias = list[float]`），参数与返回值全部注解；可能返回 `None` 的写 `X | None`。
- Docstring 用 Google 风格：一句话摘要，然后 `Args:`、`Returns:`，会抛异常时加 `Raises:`，最后 `Examples:`。
- `Examples:` 写成 doctest，数值取自知识正文的示例，覆盖正常输入、边界情况（如零向量、共线）与异常三类。
- doctest 必须对任何正确实现都成立：
  - 浮点结果有舍入误差时先 `round` 再展示；
  - 结果是 int 还是 float 取决于实现时，用 `== 期望值` 比较；
  - 异常示例加 `# doctest: +IGNORE_EXCEPTION_DETAIL`。
- 交付前用一份临时参考实现（放在 scratchpad，不进仓库）跑通全部 doctest，确认示例本身正确。
- 模块 docstring 写明运行命令 `python3 -m doctest <name>.py -v`，并注明骨架状态下失败是预期结果。
- 不引入 pydantic 等运行时校验库：输入是本地的列表与标量，类型注解加 mypy 检查已足够。
