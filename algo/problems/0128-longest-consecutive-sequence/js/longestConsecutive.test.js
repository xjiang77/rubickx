const { longestConsecutive } = require('./longestConsecutive');

test('basic', () => { expect(longestConsecutive([100, 4, 200, 1, 3, 2])).toBe(4); });
test('with duplicates', () => { expect(longestConsecutive([0, 3, 7, 2, 5, 8, 4, 6, 0, 1])).toBe(9); });
test('empty', () => { expect(longestConsecutive([])).toBe(0); });
test('single', () => { expect(longestConsecutive([5])).toBe(1); });
