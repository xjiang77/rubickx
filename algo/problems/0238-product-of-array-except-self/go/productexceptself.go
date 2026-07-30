package productexceptself

// ProductExceptSelf 前缀积×后缀积两趟遍历:res 先存左侧乘积,再原地乘右侧乘积。
// 不用除法(含 0 会崩)。O(n) 时间,除输出外 O(1) 空间。
func ProductExceptSelf(nums []int) []int {
	res := make([]int, len(nums))
	prefix := 1
	for i, x := range nums {
		res[i] = prefix
		prefix *= x
	}
	suffix := 1
	for i := len(nums) - 1; i >= 0; i-- {
		res[i] *= suffix
		suffix *= nums[i]
	}
	return res
}
