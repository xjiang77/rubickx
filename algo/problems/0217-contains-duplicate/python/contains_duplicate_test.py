import os, sys
sys.path.insert(0, os.path.dirname(__file__))
from contains_duplicate import contains_duplicate


def test_has_dup():
    assert contains_duplicate([1, 2, 3, 1]) is True

def test_unique():
    assert contains_duplicate([1, 2, 3, 4]) is False

def test_empty():
    assert contains_duplicate([]) is False
