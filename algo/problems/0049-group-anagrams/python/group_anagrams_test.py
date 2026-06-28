import os, sys
sys.path.insert(0, os.path.dirname(__file__))
from group_anagrams import group_anagrams


def _norm(groups):
    return sorted(sorted(g) for g in groups)


def test_basic():
    res = group_anagrams(["eat", "tea", "tan", "ate", "nat", "bat"])
    assert _norm(res) == _norm([["bat"], ["nat", "tan"], ["ate", "eat", "tea"]])

def test_empty_string():
    assert _norm(group_anagrams([""])) == [[""]]

def test_single():
    assert _norm(group_anagrams(["a"])) == [["a"]]
