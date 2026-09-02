const { groupAnagrams } = require('./groupAnagrams');

function norm(groups) {
  return groups
    .map((g) => [...g].sort())
    .sort((a, b) => a.join(',').localeCompare(b.join(',')));
}

test('basic', () => {
  const res = groupAnagrams(['eat', 'tea', 'tan', 'ate', 'nat', 'bat']);
  expect(norm(res)).toEqual(norm([['bat'], ['nat', 'tan'], ['ate', 'eat', 'tea']]));
});
test('empty string', () => { expect(norm(groupAnagrams(['']))).toEqual([['']]); });
test('single', () => { expect(norm(groupAnagrams(['a']))).toEqual([['a']]); });
