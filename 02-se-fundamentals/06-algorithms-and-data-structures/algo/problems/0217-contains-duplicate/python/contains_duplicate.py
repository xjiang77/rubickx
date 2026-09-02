def contains_duplicate(nums: list[int]) -> bool:
    """集合去重：边遍历边查，命中重复立即返回。O(n)/O(n)。
    一行写法：return len(set(nums)) != len(nums)（牺牲提前退出）。"""
    seen: set[int] = set()
    for x in nums:
        if x in seen:
            return True
        seen.add(x)
    return False
