import os, sys
sys.path.insert(0, os.path.dirname(__file__))
from top_k_frequent import top_k_frequent


def test_basic():
    assert sorted(top_k_frequent([1, 1, 1, 2, 2, 3], 2)) == [1, 2]

def test_single():
    assert top_k_frequent([1], 1) == [1]

def test_same_freq():
    assert sorted(top_k_frequent([4, 5, 6], 3)) == [4, 5, 6]
