/**
 * 以"排序后的字符串"为键，用 Map 聚合。O(n·k·log k)。
 * @param {string[]} strs
 * @returns {string[][]}
 */
function groupAnagrams(strs) {
  const groups = new Map();
  for (const s of strs) {
    const key = s.split('').sort().join('');
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(s);
  }
  return [...groups.values()];
}

module.exports = { groupAnagrams };
