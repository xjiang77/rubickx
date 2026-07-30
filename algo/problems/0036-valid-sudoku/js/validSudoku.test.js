const { isValidSudoku } = require('./validSudoku');

const VALID = [
  '53..7....',
  '6..195...',
  '.98....6.',
  '8...6...3',
  '4..8.3..1',
  '7...2...6',
  '.6....28.',
  '...419..5',
  '....8..79',
];

const toBoard = (rows) => rows.map((row) => row.split(''));

test('valid board', () => { expect(isValidSudoku(toBoard(VALID))).toBe(true); });

test('box duplicate', () => {
  const rows = ['8' + VALID[0].slice(1), ...VALID.slice(1)]; // (0,0) 5→8,与 (2,2) 的 8 同 box
  expect(isValidSudoku(toBoard(rows))).toBe(false);
});

test('row duplicate', () => {
  const rows = [VALID[0].slice(0, 2) + '5' + VALID[0].slice(3), ...VALID.slice(1)];
  expect(isValidSudoku(toBoard(rows))).toBe(false);
});
