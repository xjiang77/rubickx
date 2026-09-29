"""notebook 里的展示助手：颜色色块、实现状态检查。只用标准库与 IPython.display。"""

from IPython.display import HTML


def color_swatches(*items):
    """把若干 (rgb, 标签) 显示为一排色块，色块之间用 items 里的运算符字符串隔开。

    用法：color_swatches((red, "红 × 0.5"), "+", (yellow, "黄 × 0.5"), "=", (orange, "橙"))
    """
    parts = []
    for item in items:
        if isinstance(item, str):
            parts.append(f'<span style="font-size:28px;margin:0 6px">{item}</span>')
            continue
        rgb, label = item
        r, g, b = (round(x, 1) for x in rgb)
        parts.append(
            '<div style="display:inline-block;margin:6px;text-align:center">'
            f'<div style="width:110px;height:60px;border-radius:8px;background:rgb({r},{g},{b})"></div>'
            f'{label}<br><small>({r}, {g}, {b})</small></div>'
        )
    return HTML("".join(parts))


def report_implemented(module, checks):
    """打印 module 里哪些函数已实现。checks 是 {函数名: 示例参数元组}；调用抛 NotImplementedError 的算未实现。"""
    pending = []
    for name, args in checks.items():
        try:
            getattr(module, name)(*args)
        except NotImplementedError:
            pending.append(name)
        except Exception:
            pass  # 实现有错也算"已实现"，交给验证单元格报告
    if pending:
        print("待实现：", ", ".join(pending), "→ 打开 vectors.py 编写后重新执行本单元格")
    else:
        print("已实现：", ", ".join(checks))
