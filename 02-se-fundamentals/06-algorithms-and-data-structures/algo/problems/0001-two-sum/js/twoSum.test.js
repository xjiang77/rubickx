const { twoSum } = require('./twoSum');

test('basic', () => { expect(twoSum([2, 7, 11, 15], 9)).toEqual([0, 1]); });
test('middle', () => { expect(twoSum([3, 2, 4], 6)).toEqual([1, 2]); });
test('same value', () => { expect(twoSum([3, 3], 6)).toEqual([0, 1]); });
