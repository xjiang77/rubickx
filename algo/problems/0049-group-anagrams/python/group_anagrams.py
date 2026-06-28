from collections import defaultdict


def group_anagrams(strs: list[str]) -> list[list[str]]:
    """以"排序后的字符串"为键聚合。O(n·k·log k)。
    更优键：用 26 维字符计数元组作键，可降到 O(n·k)。"""
    groups: dict[str, list[str]] = defaultdict(list)
    for s in strs:
        key = "".join(sorted(s))
        groups[key].append(s)
    return list(groups.values())
