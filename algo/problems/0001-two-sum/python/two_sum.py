def two_sum(nums: list[int], target: int) -> list[int]:
    """一次遍历 + 哈希表：seen[已见过的值] = 它的下标。
    命中 target-x 即返回。时间 O(n)、空间 O(n)。"""
    seen: dict[int, int] = {}
    for i, x in enumerate(nums):
        need = target - x
        if need in seen:
            return [seen[need], i]
        seen[x] = i
    return []
