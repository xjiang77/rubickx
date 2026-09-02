const { productExceptSelf } = require('./productExceptSelf');

// JS 数字是 float64,乘法会产生 -0(如 1*0*-3*3 = -0)。
// jest 的 toEqual 按 Object.is 区分 -0 与 0,先 +0 归一化再比较。
const normalize = (arr) => arr.map((x) => x + 0);

test('basic', () => { expect(productExceptSelf([1, 2, 3, 4])).toEqual([24, 12, 8, 6]); });
test('with zero', () => { expect(normalize(productExceptSelf([-1, 1, 0, -3, 3]))).toEqual([0, 0, 9, 0, 0]); });
test('two elements', () => { expect(productExceptSelf([2, 3])).toEqual([3, 2]); });
