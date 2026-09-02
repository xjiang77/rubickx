def product_except_self(nums: list[int]) -> list[int]:
    """前缀积 × 后缀积:res 先存左侧乘积,再从右往左原地乘上右侧乘积。
    禁用除法(题目要求;且数组含 0 时除法方案会崩)。时间 O(n),额外空间 O(1)。"""
    n = len(nums)
    res = [1] * n
    prefix = 1
    for i in range(n):
        res[i] = prefix
        prefix *= nums[i]
    suffix = 1
    for i in range(n - 1, -1, -1):
        res[i] *= suffix
        suffix *= nums[i]
    return res
