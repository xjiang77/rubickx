/**
 * 字符计数：用 Map 统计 s，再用 t 抵消。O(n)/O(k)。
 * @param {string} s
 * @param {string} t
 * @returns {boolean}
 */
function isAnagram(s, t) {
  if (s.length !== t.length) return false;
  const count = new Map();
  for (const c of s) count.set(c, (count.get(c) ?? 0) + 1);
  for (const c of t) {
    const n = count.get(c);
    if (!n) return false; // undefined 或 0 都说明 t 多出了字符
    count.set(c, n - 1);
  }
  return true;
}

module.exports = { isAnagram };
