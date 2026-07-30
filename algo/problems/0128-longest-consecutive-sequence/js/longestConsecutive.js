/**
 * Set 去重,只从序列起点(x-1 不在集合)向右延伸计数。
 * 每个元素最多访问两次,均摊 O(n)/O(n)。
 * @param {number[]} nums
 * @returns {number}
 */
function longestConsecutive(nums) {
  const set = new Set(nums);
  let best = 0;
  for (const x of set) {
    if (set.has(x - 1)) continue; // 不是起点
    let len = 1;
    while (set.has(x + len)) len++;
    best = Math.max(best, len);
  }
  return best;
}

module.exports = { longestConsecutive };
