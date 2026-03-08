#!/usr/bin/env python3

import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from harness.run import grade_run, init_run


class HarnessTest(unittest.TestCase):
    def test_init_run_creates_case_workspace(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_dir = Path(tmpdir) / "run"
            init_run(run_dir, ["git-pro-book"], overwrite=False)

            self.assertTrue((run_dir / "manifest.json").exists())
            self.assertTrue((run_dir / "git-pro-book" / "input" / "task.md").exists())
            self.assertTrue((run_dir / "git-pro-book" / "output" / "result.json").exists())
            self.assertTrue(
                (run_dir / "git-pro-book" / "output" / "interactive_course_zh.json").exists()
            )

    def test_grade_run_scores_completed_case(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            run_dir = Path(tmpdir) / "run"
            init_run(run_dir, ["git-pro-book", "se-radio-legacy-code"], overwrite=False)

            git_case_dir = run_dir / "git-pro-book" / "output"
            git_result = {
                "resource_id": "git-pro-book",
                "decision": {
                    "should_transform": True,
                    "target_format": "interactive_course",
                    "confidence": 0.95,
                    "reasoning": "这本书结构完整，但 reference 味太重，需要课程化。",
                },
                "citations": [
                    {
                        "label": "Pro Git",
                        "url": "https://git-scm.com/book/en/v2",
                        "evidence": "它反复强调 working tree、index 和 commit graph 的关系。",
                    }
                ],
                "key_takeaways": [
                    "working tree、index、commit graph 是理解 Git 的核心心智模型。",
                    "branch 本质上是可移动指针，因此 merge 和 rebase 的成本与风险不同。",
                    "remote collaboration 需要先理解 divergence，再决定是否 rebase 或 force push。",
                ],
            }
            (git_case_dir / "result.json").write_text(
                json.dumps(git_result, ensure_ascii=False, indent=2) + "\n",
                encoding="utf-8",
            )
            (git_case_dir / "summary_zh.md").write_text(
                """# Pro Git 中文学习摘要

## 这份材料是什么

这是一份系统讲解 Git 的经典材料，重点不是背命令，而是建立 working tree、index、commit graph 的模型。

## 适合谁

适合已经会用 Git，但经常在 branch、merge、rebase、remote collaboration 上靠经验碰运气的工程师。

## 核心知识点

- working tree、index、commit graph 决定了大多数命令的行为。
- branch 是指针，因此 merge 和 rebase 是两种不同的历史组织策略。
- remote collaboration 需要理解 fetch、pull、push 以及分叉历史。

## 建议学习路径

先掌握 working tree 和 index，再学习 branch、merge、rebase，最后进入 remote collaboration 与团队协作。

## 引用

- https://git-scm.com/book/en/v2
""",
                encoding="utf-8",
            )
            (git_case_dir / "interactive_course_zh.json").write_text(
                json.dumps(
                    {
                        "title": "Git 心智模型入门课",
                        "resource_id": "git-pro-book",
                        "language": "zh-CN",
                        "lessons": [
                            {
                                "title": "理解 working tree、index、commit graph",
                                "goal": "建立 Git 的核心模型。",
                                "exercise": "解释一次 add/commit 为什么会改变不同层。",
                                "checkpoint": "能画出三个区域的关系图。",
                            },
                            {
                                "title": "branch、merge 与 rebase",
                                "goal": "理解历史如何被组织。",
                                "exercise": "比较 merge 与 rebase 对 commit graph 的影响。",
                                "checkpoint": "能说明什么时候不该 rebase 公共历史。",
                            },
                            {
                                "title": "remote collaboration 实战",
                                "goal": "把个人操作升级成团队协作。",
                                "exercise": "模拟 fetch、pull、push 冲突后的处理。",
                                "checkpoint": "能解释 divergence 与 force push 风险。",
                            },
                        ],
                    },
                    ensure_ascii=False,
                    indent=2,
                )
                + "\n",
                encoding="utf-8",
            )
            (git_case_dir / "trace.jsonl").write_text(
                "\n".join(
                    [
                        json.dumps({"event": "read_case"}),
                        json.dumps({"event": "draft_summary"}),
                        json.dumps({"event": "write_course"}),
                    ]
                )
                + "\n",
                encoding="utf-8",
            )

            podcast_case_dir = run_dir / "se-radio-legacy-code" / "output"
            podcast_result = {
                "resource_id": "se-radio-legacy-code",
                "decision": {
                    "should_transform": True,
                    "target_format": "blog_post",
                    "confidence": 0.89,
                    "reasoning": "podcast 更适合整理成可快速阅读的中文 blog。",
                },
                "citations": [
                    {
                        "label": "SE Radio",
                        "url": "https://www.se-radio.net/2020/09/episode-429-michael-feathers-on-working-effectively-with-legacy-code/",
                        "evidence": "讨论围绕 legacy code、safety net、seams 和 feedback loop 展开。",
                    }
                ],
                "key_takeaways": [
                    "legacy code 的难点是无法安全修改，而不是代码旧。",
                    "先建立 safety net，再做 small refactor。",
                    "seams 和 tests 能缩短 feedback loop。",
                ],
            }
            (podcast_case_dir / "result.json").write_text(
                json.dumps(podcast_result, ensure_ascii=False, indent=2) + "\n",
                encoding="utf-8",
            )
            (podcast_case_dir / "summary_zh.md").write_text(
                """# Legacy Code 中文学习摘要

## 这份材料是什么

这是一期聚焦 legacy code 的 podcast，对象是需要在复杂旧系统里持续交付的工程师。

## 适合谁

适合经常在没有 tests、依赖耦合严重、很难安全改动的系统里工作的团队。

## 核心知识点

- legacy code 的本质是缺少安全修改能力。
- safety net 可以来自 tests、instrumentation、seams。
- 应优先做可回滚、可反馈的 small refactor。

## 建议学习路径

先识别 feedback loop，再寻找 seams，最后补 tests 并做小步修改。

## 引用

- https://www.se-radio.net/2020/09/episode-429-michael-feathers-on-working-effectively-with-legacy-code/
""",
                encoding="utf-8",
            )
            (podcast_case_dir / "blog_post_zh.md").write_text(
                """# 遗留系统不是旧，而是难以安全修改

## 为什么值得学

很多团队卡住，不是因为没人懂新架构，而是没有 safety net，所以每次改 legacy code 都像赌博。

## 三个最重要的观点

- 先建立 feedback loop，再谈大改。
- seams 让 small refactor 成为可能。
- tests 不是形式主义，而是把恐惧变成可验证过程。

## 如何把它用到实际工作

先找一个最小变更点，补最短路径的 tests 或 instrumentation，再做一次 small refactor，观察反馈并重复。

## 引用

- https://www.se-radio.net/2020/09/episode-429-michael-feathers-on-working-effectively-with-legacy-code/
""",
                encoding="utf-8",
            )
            (podcast_case_dir / "trace.jsonl").write_text(
                "\n".join(
                    [
                        json.dumps({"event": "read_case"}),
                        json.dumps({"event": "decide_blog"}),
                        json.dumps({"event": "write_blog"}),
                    ]
                )
                + "\n",
                encoding="utf-8",
            )

            grade_run(run_dir)
            report = json.loads((run_dir / "report.json").read_text(encoding="utf-8"))
            self.assertGreaterEqual(report["average_score"], 80)
            self.assertEqual(len(report["cases"]), 2)


if __name__ == "__main__":
    unittest.main()
