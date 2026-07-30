def longest_consecutive(nums: list[int]) -> int:
    """set 去重后,只从"序列起点"(x-1 不在集合)开始向右延伸计数。
    每个元素最多被访问两次,均摊 O(n) 时间、O(n) 空间。"""
    s = set(nums)
    best = 0
    for x in s:
        if x - 1 in s:  # 不是起点,跳过——这是 O(n) 的关键
            continue
        length = 1
        while x + length in s:
            length += 1
        best = max(best, length)
    return best
