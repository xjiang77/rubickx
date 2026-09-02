const { topKFrequent } = require('./topKFrequent');

const asc = (a, b) => a - b;

test('basic', () => { expect(topKFrequent([1, 1, 1, 2, 2, 3], 2).sort(asc)).toEqual([1, 2]); });
test('single', () => { expect(topKFrequent([1], 1)).toEqual([1]); });
test('same freq', () => { expect(topKFrequent([4, 5, 6], 3).sort(asc)).toEqual([4, 5, 6]); });
