/**
 * 前缀积 × 后缀积:res 先存左侧乘积,再从右往左原地乘右侧乘积。
 * 不用除法(含 0 会崩)。O(n) 时间,除输出外 O(1) 空间。
 * @param {number[]} nums
 * @returns {number[]}
 */
function productExceptSelf(nums) {
  const n = nums.length;
  const res = new Array(n).fill(1);
  let prefix = 1;
  for (let i = 0; i < n; i++) {
    res[i] = prefix;
    prefix *= nums[i];
  }
  let suffix = 1;
  for (let i = n - 1; i >= 0; i--) {
    res[i] *= suffix;
    suffix *= nums[i];
  }
  return res;
}

module.exports = { productExceptSelf };
