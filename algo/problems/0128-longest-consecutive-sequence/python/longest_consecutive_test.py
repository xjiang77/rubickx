import os, sys
sys.path.insert(0, os.path.dirname(__file__))
from longest_consecutive import longest_consecutive


def test_basic():
    assert longest_consecutive([100, 4, 200, 1, 3, 2]) == 4

def test_with_duplicates():
    assert longest_consecutive([0, 3, 7, 2, 5, 8, 4, 6, 0, 1]) == 9

def test_empty():
    assert longest_consecutive([]) == 0

def test_single():
    assert longest_consecutive([5]) == 1
