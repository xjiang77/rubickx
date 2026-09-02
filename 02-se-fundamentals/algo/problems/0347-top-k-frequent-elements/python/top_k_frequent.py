from collections import Counter


def top_k_frequent(nums: list[int], k: int) -> list[int]:
    """计数 + 桶排序:频次最大为 n,把数字按频次放进桶,从高频桶往低频收集 k 个。
    时间 O(n)、空间 O(n),优于堆解法的 O(n log k)。"""
    counts = Counter(nums)
    buckets: list[list[int]] = [[] for _ in range(len(nums) + 1)]
    for x, c in counts.items():
        buckets[c].append(x)
    res: list[int] = []
    for c in range(len(buckets) - 1, 0, -1):
        for x in buckets[c]:
            res.append(x)
            if len(res) == k:
                return res
    return res
