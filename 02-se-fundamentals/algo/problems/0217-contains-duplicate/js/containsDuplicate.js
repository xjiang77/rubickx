/**
 * 用 Set 边遍历边查，命中重复立即返回。O(n)/O(n)。
 * 一行写法：return new Set(nums).size !== nums.length（牺牲提前退出）。
 * @param {number[]} nums
 * @returns {boolean}
 */
function containsDuplicate(nums) {
  const seen = new Set();
  for (const x of nums) {
    if (seen.has(x)) return true;
    seen.add(x);
  }
  return false;
}

module.exports = { containsDuplicate };
