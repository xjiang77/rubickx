# Go Sprint 算法线题单(2026-07-27 ~ 09-20)

配套 vault:`[[05 - Go Sprint 8-Week Plan]]` 线 B。概念主线《Hello 算法》,范式参考 labuladong,全部用 Go 解。
题解放 `problems/NNNN-slug/go/`(沿用现有多语言目录约定,其余语言列留 ⬜);手写结构放 `structures/go/`。
完成状态记入 `PROGRESS.md`(✅ 需测试通过)。已完成的 1/217/242/49 不重复计入。

## W1 双指针 / 滑动窗口(12)

125 Valid Palindrome · 167 Two Sum II · 15 3Sum · 11 Container With Most Water · 42 Trapping Rain Water(stretch)· 121 Best Time to Buy and Sell Stock · 3 Longest Substring Without Repeating · 424 Longest Repeating Character Replacement · 567 Permutation in String · 76 Minimum Window Substring · 239 Sliding Window Maximum · 128 Longest Consecutive Sequence

## W2 链表 / 栈队列 / 单调栈(12)

206 Reverse Linked List · 141 Linked List Cycle · 21 Merge Two Sorted Lists · 143 Reorder List · 19 Remove Nth Node · 138 Copy List with Random Pointer · 2 Add Two Numbers · 23 Merge k Sorted Lists · 146 LRU Cache(用自己的 structures/go/lru)· 20 Valid Parentheses · 155 Min Stack · 739 Daily Temperatures

## W3 二分 / 树 / 堆(12)

704 Binary Search · 74 Search a 2D Matrix · 33 Search in Rotated Sorted Array · 153 Find Minimum in Rotated Sorted Array · 875 Koko Eating Bananas · 226 Invert Binary Tree · 104 Maximum Depth · 110 Balanced Binary Tree · 98 Validate BST · 235 LCA of BST · 102 Level Order Traversal · 105 Construct from Preorder & Inorder;堆:215 Kth Largest · 347 Top K(已有 Py 解,补 Go)· 295 Find Median from Data Stream(stretch)

## W4 回溯 / Trie / 并查集(12)

78 Subsets · 90 Subsets II · 46 Permutations · 39 Combination Sum · 40 Combination Sum II · 17 Letter Combinations · 79 Word Search · 51 N-Queens · 208 Implement Trie · 211 Design Add & Search Words · 547 Number of Provinces · 684 Redundant Connection

## W5 图(10)

200 Number of Islands · 994 Rotting Oranges · 133 Clone Graph · 417 Pacific Atlantic Water Flow · 207 Course Schedule · 210 Course Schedule II · 743 Network Delay Time(Dijkstra)· 1584 Min Cost to Connect All Points · 787 Cheapest Flights Within K Stops · 127 Word Ladder(stretch)

## W6 一维 DP / 背包(10)

70 Climbing Stairs · 198 House Robber · 213 House Robber II · 5 Longest Palindromic Substring · 91 Decode Ways · 322 Coin Change · 300 Longest Increasing Subsequence · 139 Word Break · 416 Partition Equal Subset Sum · 494 Target Sum

## W7 二维/区间 DP / 贪心 / 区间(10)

62 Unique Paths · 1143 Longest Common Subsequence · 72 Edit Distance · 97 Interleaving String(stretch)· 53 Maximum Subarray · 55 Jump Game · 45 Jump Game II · 56 Merge Intervals · 435 Non-overlapping Intervals · 986 Interval List Intersections

## W8 收尾

错题全部重做(不看旧解);计时 mock ×2(90min / 2 medium,coach 出题)。

---

累计 ~88 题 + stretch。每题固定流程:5 分钟内说出暴力解与目标复杂度 → 写 Go 解 + 至少 3 个自测 case → 对照最优解,把差异写进 NOTES.md 的一句话里。卡 25 分钟看提示,卡 40 分钟看题解并标记为错题。
