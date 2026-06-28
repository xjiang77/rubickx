const { isAnagram } = require('./validAnagram');

test('true', () => { expect(isAnagram('anagram', 'nagaram')).toBe(true); });
test('false', () => { expect(isAnagram('rat', 'car')).toBe(false); });
test('diff len', () => { expect(isAnagram('a', 'ab')).toBe(false); });
