# rubickx Harness

这个 harness 不是 generic benchmark，而是把 `rubickx` 的真实目标转成可重复跑的 content-production loop：

1. intake 一个经典学习材料
2. 判断它值不值得沉淀
3. 生成中文学习摘要
4. 必要时转成 interactive course 或 blog post
5. 记录 trace，方便复盘失败模式

它参考了两篇文章的共同实践：

- OpenAI 的 harness engineering：优先评测真实 end-to-end task，而不是孤立 prompt 技巧。
- LangChain 的 harness engineering：保留 trajectory / trace，做 partial-credit grading，用 failure taxonomy 驱动下一轮改进。

参考链接：

- https://openai.com/index/harness-engineering/
- https://blog.langchain.com/improving-deep-agents-with-harness-engineering/

## 目录

```text
harness/
├── cases/           # deterministic task specs
├── fixtures/        # frozen source snapshots
├── runs/            # generated runs (gitignored)
└── run.py           # init-run / grade / list-cases
```

## 设计原则

- Real task first：case 模拟真实内容策展和课程化工作，而不是 toy prompt。
- Frozen input：用本地 snapshot 固定输入，避免网络波动让 eval 漂移。
- Artifact contract：每个 case 都要求 `result.json`、`summary_zh.md`、主产物、`trace.jsonl`。
- Partial credit：grader 会分别给 artifact contract、grounding、decision fit、Chinese output、pedagogy、traceability 打分。
- Failure taxonomy：报告直接输出 `missing_artifact`、`wrong_transform`、`weak_grounding` 等失败标签。

## 命令

列出 case：

```bash
python3 -m harness.run list-cases
```

初始化一个 run：

```bash
python3 -m harness.run init-run --run-dir harness/runs/manual
```

只初始化单个 case：

```bash
python3 -m harness.run init-run --run-dir harness/runs/git-only --case git-pro-book
```

评分：

```bash
python3 -m harness.run grade --run-dir harness/runs/manual
```

## 当前 case

- `git-pro-book`: book/site -> `interactive_course`
- `system-design-primer`: GitHub repo -> `interactive_course`
- `mit-6-824-lecture-1`: YouTube lecture -> `interactive_course`
- `se-radio-legacy-code`: podcast -> `blog_post`

## 产物约定

`result.json` 至少包含：

```json
{
  "resource_id": "git-pro-book",
  "decision": {
    "should_transform": true,
    "target_format": "interactive_course",
    "confidence": 0.93,
    "reasoning": "..."
  },
  "citations": [
    {
      "label": "Pro Git",
      "url": "https://git-scm.com/book/en/v2",
      "evidence": "..."
    }
  ],
  "key_takeaways": ["...", "...", "..."]
}
```

这版 grader 先用 deterministic heuristics，不依赖 LLM judge。后面如果要接更细的 style / factuality judgment，可以在这个 contract 上继续扩。
