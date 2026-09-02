const { containsDuplicate } = require('./containsDuplicate');

test('has dup', () => { expect(containsDuplicate([1, 2, 3, 1])).toBe(true); });
test('unique', () => { expect(containsDuplicate([1, 2, 3, 4])).toBe(false); });
test('empty', () => { expect(containsDuplicate([])).toBe(false); });
