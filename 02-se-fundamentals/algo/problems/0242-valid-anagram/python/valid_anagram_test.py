import os, sys
sys.path.insert(0, os.path.dirname(__file__))
from valid_anagram import is_anagram


def test_true():
    assert is_anagram("anagram", "nagaram") is True

def test_false():
    assert is_anagram("rat", "car") is False

def test_diff_len():
    assert is_anagram("a", "ab") is False
