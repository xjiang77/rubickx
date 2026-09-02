/**
 * 计数 + 桶排序:Map 计数,下标=频次的桶数组,从高频往低频收集 k 个。O(n)/O(n)。
 * @param {number[]} nums
 * @param {number} k
 * @returns {number[]}
 */
function topKFrequent(nums, k) {
  const counts = new Map();
  for (const x of nums) counts.set(x, (counts.get(x) ?? 0) + 1);
  const buckets = Array.from({ length: nums.length + 1 }, () => []);
  for (const [x, c] of counts) buckets[c].push(x);
  const res = [];
  for (let c = buckets.length - 1; c > 0 && res.length < k; c--) {
    res.push(...buckets[c]);
  }
  return res.slice(0, k);
}

module.exports = { topKFrequent };
