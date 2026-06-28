from collections import Counter


def is_anagram(s: str, t: str) -> bool:
    """字符计数相等即互为字母异位词。O(n)/O(k)，k=字符集大小。
    Counter 让它变成一行：先比长度（快速剪枝），再比计数。"""
    if len(s) != len(t):
        return False
    return Counter(s) == Counter(t)
