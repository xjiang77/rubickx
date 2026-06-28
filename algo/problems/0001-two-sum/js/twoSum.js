/**
 * 一次遍历 + Map：seen.get(值)=下标，命中 target-x 即返回。O(n)/O(n)。
 * @param {number[]} nums
 * @param {number} target
 * @returns {number[]}
 */
function twoSum(nums, target) {
  const seen = new Map();
  for (let i = 0; i < nums.length; i++) {
    const need = target - nums[i];
    if (seen.has(need)) return [seen.get(need), i];
    seen.set(nums[i], i);
  }
  return [];
}

module.exports = { twoSum };
