# 04 — Shaping the build

对应 Skills Map 第四项技能：coding agent 拿到清楚的 spec 就能交付，工程师的工作转向决定 spec 里该有什么——产品感、业务背景、何时快速做 MVP、何时放慢。

本目录放"决定造什么"的可执行证据：判断材料值不值得沉淀、该转成什么形态的 harness；后续的 spec 与 ADR 也归这里。

| Track | 内容 | 验证 |
| --- | --- | --- |
| `harness/` | 内容策展决策 harness：4 个 deterministic case（git-pro-book、mit-6-824-lecture-1、se-radio-legacy-code、system-design-primer），每个 case 要求 agent 判断"值不值得沉淀、转成课程还是 blog"，grader 按 artifact contract、grounding、decision fit、中文输出、pedagogy、traceability 打分并输出 failure taxonomy | 根目录 `make test-harness`；`make harness-list` / `harness-init` / `harness-grade` |

harness 的 grader 部分与第一项技能的 evals 有重叠；放在这里是因为 case 的核心是决策，不是模型行为。
