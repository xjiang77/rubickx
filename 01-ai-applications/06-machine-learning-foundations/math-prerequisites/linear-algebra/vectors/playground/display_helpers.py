"""notebook 里的展示助手：颜色色块、实现状态检查。只用标准库；IPython 只在显示时导入。"""
from types import ModuleType
from typing import Any

Rgb = tuple[float, float, float]


def swatch_html(rgb: Rgb, label: str) -> str:
    """一个色块的 HTML：上面是颜色，下面是标签与分量。

    Examples:
        >>> "rgb(255,0,0)" in swatch_html((255, 0, 0), "红")
        True
    """
    r, g, b = (round(x, 1) for x in rgb)
    return (
        '<div style="display:inline-block;margin:6px;text-align:center">'
        f'<div style="width:110px;height:60px;border-radius:8px;background:rgb({r},{g},{b})"></div>'
        f"{label}<br><small>({r}, {g}, {b})</small></div>"
    )


def color_swatches(*items: tuple[Rgb, str] | str) -> Any:
    """把若干 (rgb, 标签) 显示为一排色块；字符串项作为色块之间的运算符。

    用法：``color_swatches((red, "红 × 0.5"), "+", (yellow, "黄 × 0.5"), "=", (orange, "橙"))``

    Returns:
        IPython 的 HTML 对象，作为单元格最后一个表达式时自动显示。
    """
    from IPython.display import HTML  # 延迟导入：没有 IPython 的环境也能导入本模块

    parts = []
    for item in items:
        if isinstance(item, str):
            parts.append(f'<span style="font-size:28px;margin:0 6px">{item}</span>')
        else:
            rgb, label = item
            parts.append(swatch_html(rgb, label))
    return HTML("".join(parts))


def pending_functions(module: ModuleType, checks: dict[str, tuple]) -> list[str]:
    """返回 checks 中仍抛 NotImplementedError 的函数名。

    Args:
        module: 练习模块，如 vectors。
        checks: {函数名: 示例参数元组}，用示例参数调用一次来探测。

    Returns:
        未实现的函数名，按 checks 的顺序。实现有错误（抛其他异常）的算已实现，交给验证单元格报告。

    Examples:
        >>> import types
        >>> m = types.ModuleType("m")
        >>> m.done = lambda x: x
        >>> def todo(x): raise NotImplementedError
        >>> m.todo = todo
        >>> pending_functions(m, {"done": (1,), "todo": (1,)})
        ['todo']
    """
    pending = []
    for name, args in checks.items():
        try:
            getattr(module, name)(*args)
        except NotImplementedError:
            pending.append(name)
        except Exception:
            pass
    return pending


def report_implemented(module: ModuleType, checks: dict[str, tuple]) -> None:
    """打印 module 里哪些函数已实现、哪些待实现。"""
    pending = pending_functions(module, checks)
    if pending:
        print("待实现：", ", ".join(pending), "→ 打开 vectors.py 编写后重新执行本单元格")
    else:
        print("已实现：", ", ".join(checks))
