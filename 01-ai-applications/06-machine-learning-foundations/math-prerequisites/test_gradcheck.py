"""数学前置的可运行核对：python3 test_gradcheck.py"""
import unittest

from gradcheck import analytic_gradient, chain_factors, descend, loss, numeric_derivative, numeric_gradient

ONE = [(2.0, 10.0)]
TWO = [(2.0, 10.0), (1.0, 4.0)]


class Derivative(unittest.TestCase):
    def test_square_error_slope(self):
        # L(e) = e²，在 e = −4 处导数为 −8
        self.assertAlmostEqual(numeric_derivative(lambda e: e * e, -4.0), -8.0, places=6)

    def test_local_linear_approximation(self):
        # e 从 −4 改到 −3.99：实际变化 −0.0799，线性近似 −0.08
        actual = (-3.99) ** 2 - 16
        self.assertAlmostEqual(actual, -0.0799, places=9)
        self.assertAlmostEqual(actual, -8 * 0.01, places=3)


class ChainRule(unittest.TestCase):
    def test_factors_multiply(self):
        f = chain_factors(3.0, 2.0, 10.0)
        self.assertEqual(f, {"dL/de": -8.0, "de/dyhat": 1.0, "dyhat/dw": 2.0})
        self.assertEqual(f["dL/de"] * f["de/dyhat"] * f["dyhat/dw"], -16.0)

    def test_matches_numeric(self):
        g = analytic_gradient(3.0, 0.0, ONE)
        n = numeric_gradient(3.0, 0.0, ONE)
        self.assertAlmostEqual(g["w"], -16.0)
        self.assertAlmostEqual(n["w"], g["w"], places=5)

    def test_finite_step(self):
        # w 从 3 改到 3.01：L 从 16 变为 15.8404
        self.assertAlmostEqual(loss(3.01, 0.0, ONE), 15.8404, places=9)

    def test_paths_add(self):
        # 同一个 w 作用于两个样本：−16 + (−2) = −18
        g = analytic_gradient(3.0, 0.0, TWO)
        self.assertAlmostEqual(g["w"], -18.0)
        self.assertAlmostEqual(numeric_gradient(3.0, 0.0, TWO)["w"], -18.0, places=5)


class Gradient(unittest.TestCase):
    def test_two_parameters(self):
        g = analytic_gradient(3.0, 0.0, ONE)
        self.assertEqual((g["w"], g["b"]), (-16.0, -8.0))
        n = numeric_gradient(3.0, 0.0, ONE)
        self.assertAlmostEqual(n["b"], -8.0, places=5)

    def test_descent_two_parameters(self):
        (_, _, L0), (w1, b1, L1) = descend(3.0, 0.0, ONE, 0.05, 1)
        self.assertEqual(L0, 16.0)
        self.assertAlmostEqual(w1, 3.8)
        self.assertAlmostEqual(b1, 0.4)
        self.assertAlmostEqual(L1, 4.0)

    def test_descent_weight_only_shrinks_by_036(self):
        Ls = [L for _, _, L in descend(3.0, 0.0, ONE, 0.05, 4, train_bias=False)]
        self.assertAlmostEqual(Ls[1], 5.76)
        for a, b in zip(Ls, Ls[1:]):
            self.assertAlmostEqual(b / a, 0.36)

    def test_learning_rate_too_large(self):
        # η = 0.25：w 在 3 和 7 之间来回，损失停在 16；η = 0.3：发散
        h = descend(3.0, 0.0, ONE, 0.25, 2, train_bias=False)
        self.assertEqual([round(w, 6) for w, _, _ in h], [3.0, 7.0, 3.0])
        self.assertEqual({L for _, _, L in h}, {16.0})
        Ls = [L for _, _, L in descend(3.0, 0.0, ONE, 0.3, 3, train_bias=False)]
        self.assertTrue(Ls[0] < Ls[1] < Ls[2] < Ls[3])


if __name__ == "__main__":
    unittest.main()
