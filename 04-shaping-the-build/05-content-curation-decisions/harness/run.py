#!/usr/bin/env python3

import argparse
import json
import shutil
from datetime import datetime, timezone
from pathlib import Path


HARNESS_ROOT = Path(__file__).resolve().parent
CASES_DIR = HARNESS_ROOT / "cases"
FIXTURES_DIR = HARNESS_ROOT / "fixtures"


def load_cases():
    cases = {}
    for path in sorted(CASES_DIR.glob("*.json")):
        case = json.loads(path.read_text(encoding="utf-8"))
        case["_case_path"] = str(path)
        case["_fixture_path"] = str(FIXTURES_DIR / case["fixture_file"])
        cases[case["id"]] = case
    return cases


def resolve_case_ids(cases, requested_ids):
    if not requested_ids:
        return list(cases.keys())
    missing = [case_id for case_id in requested_ids if case_id not in cases]
    if missing:
        raise SystemExit(f"Unknown case id(s): {', '.join(missing)}")
    return requested_ids


def slugify(text):
    return "".join(ch.lower() if ch.isalnum() else "-" for ch in text).strip("-")


def count_cjk(text):
    return sum(
        1
        for ch in text
        if "\u4e00" <= ch <= "\u9fff" or "\u3400" <= ch <= "\u4dbf"
    )


def normalize_text(text):
    return " ".join(str(text).lower().split())


def target_artifact_name(case):
    target = case["expected_transform"]
    if target == "interactive_course":
        return "interactive_course_zh.json"
    if target == "blog_post":
        return "blog_post_zh.md"
    return None


def render_task(case, fixture_text):
    target = case["expected_transform"]
    artifact_name = target_artifact_name(case)
    artifact_line = (
        f"3. `output/{artifact_name}`: 按决策生成的主产物。"
        if artifact_name
        else "3. 不要求额外主产物，但如果你判断需要转成内容，也必须在 `result.json` 里说明。"
    )
    must_cover = "\n".join(f"- {item}" for item in case["must_cover"])
    return f"""# {case["title"]}

你是 rubickx 的内容 agent。你的任务不是泛泛总结，而是把经典学习材料变成可复用的中文学习资产。

## 目标

- 受众：{case["audience"]}
- 学习目标：{case["learning_goal"]}
- 期望转化：`{target}`
- 说明：{case["notes"]}

## 交付物

1. `output/result.json`: 结构化决策，必须包含 `resource_id`、`decision`、`citations`、`key_takeaways`。
2. `output/summary_zh.md`: 中文摘要，至少覆盖 source、适合人群、核心知识点、建议学习路径。
{artifact_line}
4. `output/trace.jsonl`: 记录关键步骤，便于 failure analysis。

## 必须覆盖的内容

{must_cover}

## 评分重点

- 真实任务闭环：不是单纯摘要，要体现 intake -> judgment -> delivery。
- Source grounding：需要引用 source URL，并且输出里能看出内容来自输入材料。
- Chinese-first：主输出必须以中文为主，不接受英文占主导。
- Format fit：是否真的适合转成 interactive course 或 blog。
- Traceability：失败时要能从 `trace.jsonl` 回看过程。

## 输入快照

下面是为了保持 harness 稳定性准备的 deterministic snapshot。它是对原始公开材料的压缩摘要，不是原文逐字复制。

{fixture_text}
"""


def result_template(case):
    target = case["expected_transform"]
    return {
        "resource_id": case["id"],
        "resource_type": case["resource_type"],
        "decision": {
            "should_transform": target != "none",
            "target_format": target,
            "confidence": 0.0,
            "reasoning": "",
        },
        "citations": [
            {
                "label": case["title"],
                "url": case["source"]["url"],
                "evidence": "",
            }
        ],
        "key_takeaways": [],
        "gaps": [],
        "recommended_audience": case["audience"],
    }


def summary_template(case):
    return f"""# {case["title"]} 中文学习摘要

## 这份材料是什么

-

## 适合谁

-

## 核心知识点

-

## 建议学习路径

-

## 引用

- {case["source"]["url"]}
"""


def course_template(case):
    return {
        "title": "",
        "resource_id": case["id"],
        "language": "zh-CN",
        "lessons": [
            {"title": "", "goal": "", "exercise": "", "checkpoint": ""},
            {"title": "", "goal": "", "exercise": "", "checkpoint": ""},
            {"title": "", "goal": "", "exercise": "", "checkpoint": ""},
        ],
    }


def blog_template(case):
    return f"""# {case["title"]}：中文学习版整理

## 为什么值得学

-

## 三个最重要的观点

-

## 如何把它用到实际工作

-

## 引用

- {case["source"]["url"]}
"""


def trace_template():
    return ""


def ensure_parent(path):
    path.parent.mkdir(parents=True, exist_ok=True)


def write_json(path, payload):
    ensure_parent(path)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def init_run(run_dir, selected_case_ids, overwrite):
    cases = load_cases()
    case_ids = resolve_case_ids(cases, selected_case_ids)
    if run_dir.exists() and overwrite:
        shutil.rmtree(run_dir)
    if run_dir.exists() and any(run_dir.iterdir()):
        raise SystemExit(f"Run directory already exists and is not empty: {run_dir}")

    run_dir.mkdir(parents=True, exist_ok=True)
    manifest = {
        "created_at": datetime.now(timezone.utc).isoformat(),
        "case_ids": case_ids,
        "harness_root": str(HARNESS_ROOT),
    }
    write_json(run_dir / "manifest.json", manifest)

    for case_id in case_ids:
        case = cases[case_id]
        fixture_text = Path(case["_fixture_path"]).read_text(encoding="utf-8")
        case_dir = run_dir / case_id
        input_dir = case_dir / "input"
        output_dir = case_dir / "output"
        input_dir.mkdir(parents=True, exist_ok=True)
        output_dir.mkdir(parents=True, exist_ok=True)

        write_json(input_dir / "case.json", case)
        (input_dir / "source_snapshot.md").write_text(fixture_text, encoding="utf-8")
        (input_dir / "task.md").write_text(render_task(case, fixture_text), encoding="utf-8")
        write_json(output_dir / "result.json", result_template(case))
        (output_dir / "summary_zh.md").write_text(summary_template(case), encoding="utf-8")
        artifact_name = target_artifact_name(case)
        if artifact_name == "interactive_course_zh.json":
            write_json(output_dir / artifact_name, course_template(case))
        elif artifact_name == "blog_post_zh.md":
            (output_dir / artifact_name).write_text(blog_template(case), encoding="utf-8")
        (output_dir / "trace.jsonl").write_text(trace_template(), encoding="utf-8")

    print(f"Initialized run at {run_dir}")
    print("Cases:")
    for case_id in case_ids:
        print(f"  - {case_id}")


def load_json_if_exists(path):
    if not path.exists():
        return None
    return json.loads(path.read_text(encoding="utf-8"))


def score_required_fields(result_json):
    if result_json is None:
        return 0, ["missing_artifact: result.json not found"]

    failures = []
    score = 0
    if result_json.get("resource_id"):
        score += 5
    else:
        failures.append("invalid_schema: resource_id missing")
    decision = result_json.get("decision")
    if isinstance(decision, dict):
        score += 5
        if "target_format" not in decision:
            failures.append("invalid_schema: decision.target_format missing")
        if "should_transform" not in decision:
            failures.append("invalid_schema: decision.should_transform missing")
    else:
        failures.append("invalid_schema: decision missing")
    citations = result_json.get("citations")
    if isinstance(citations, list) and citations:
        score += 5
    else:
        failures.append("invalid_schema: citations missing")
    takeaways = result_json.get("key_takeaways")
    if isinstance(takeaways, list) and len(takeaways) >= 3:
        score += 5
    else:
        failures.append("invalid_schema: key_takeaways should contain at least 3 items")
    return score, failures


def score_grounding(case, result_json, combined_text):
    score = 0
    failures = []
    citations = result_json.get("citations") if isinstance(result_json, dict) else []
    urls = {item.get("url") for item in citations if isinstance(item, dict)}
    if case["source"]["url"] in urls:
        score += 10
    else:
        failures.append("weak_grounding: source url missing from citations")

    text = normalize_text(combined_text)
    hits = sum(1 for item in case["must_cover"] if normalize_text(item) in text)
    coverage_score = min(15, hits * 3)
    score += coverage_score
    if hits < max(2, len(case["must_cover"]) // 2):
        failures.append("weak_grounding: must_cover topics are underrepresented")
    return score, failures, hits


def score_decision(case, result_json):
    score = 0
    failures = []
    if not isinstance(result_json, dict):
        return score, ["wrong_transform: result.json missing"]
    decision = result_json.get("decision") or {}
    actual = decision.get("target_format", "none")
    expected = case["expected_transform"]
    if actual == expected:
        score = 20
    elif decision.get("should_transform") is True and expected != "none":
        score = 8
        failures.append(
            f"wrong_transform: expected {expected}, got {actual or 'none'}"
        )
    else:
        failures.append(
            f"wrong_transform: expected {expected}, got {actual or 'none'}"
        )
    return score, failures


def score_chinese(summary_text, artifact_text):
    summary_cjk = count_cjk(summary_text)
    artifact_cjk = count_cjk(artifact_text)
    total = summary_cjk + artifact_cjk
    if total >= 700:
        return 20, []
    if total >= 450:
        return 14, []
    if total >= 250:
        return 8, ["weak_chinese_output: content is too short for a Chinese-first deliverable"]
    return 0, ["weak_chinese_output: main artifacts are not substantially in Chinese"]


def score_pedagogy(case, artifact_json, artifact_text):
    target = case["expected_transform"]
    failures = []
    if target == "interactive_course":
        lessons = artifact_json.get("lessons") if isinstance(artifact_json, dict) else None
        if not isinstance(lessons, list) or len(lessons) < 3:
            return 0, ["weak_pedagogy: interactive course needs at least 3 lessons"]
        complete = sum(
            1
            for lesson in lessons
            if isinstance(lesson, dict)
            and lesson.get("title")
            and lesson.get("goal")
            and lesson.get("exercise")
        )
        if complete >= 3:
            return 10, []
        return 4, ["weak_pedagogy: lessons are present but incomplete"]

    if target == "blog_post":
        headings = sum(1 for line in artifact_text.splitlines() if line.strip().startswith("## "))
        if headings >= 3 and count_cjk(artifact_text) >= 250:
            return 10, []
        return 3, ["weak_pedagogy: blog post needs stronger structure and depth"]

    return 10, []


def score_trace(trace_path):
    if not trace_path.exists():
        return 0, ["missing_trace: trace.jsonl not found"]
    lines = [line for line in trace_path.read_text(encoding="utf-8").splitlines() if line.strip()]
    if not lines:
        return 0, ["missing_trace: trace.jsonl is empty"]

    parsed = 0
    for line in lines:
        try:
            item = json.loads(line)
        except json.JSONDecodeError:
            continue
        if isinstance(item, dict) and item.get("event"):
            parsed += 1
    if parsed >= 3:
        return 5, []
    if parsed >= 1:
        return 2, ["missing_trace: trace exists but is too shallow"]
    return 0, ["missing_trace: trace entries are not valid JSON events"]


def grade_case(case, case_dir):
    output_dir = case_dir / "output"
    result_path = output_dir / "result.json"
    summary_path = output_dir / "summary_zh.md"
    artifact_name = target_artifact_name(case)
    artifact_path = output_dir / artifact_name if artifact_name else None
    trace_path = output_dir / "trace.jsonl"

    result_json = load_json_if_exists(result_path)
    artifact_json = load_json_if_exists(artifact_path) if artifact_path and artifact_path.suffix == ".json" else None
    summary_text = summary_path.read_text(encoding="utf-8") if summary_path.exists() else ""
    artifact_text = ""
    if artifact_path and artifact_path.exists():
        artifact_text = artifact_path.read_text(encoding="utf-8")

    scores = {}
    failures = []

    scores["artifact_contract"], extra = score_required_fields(result_json)
    failures.extend(extra)

    combined_text = "\n".join(
        [
            summary_text,
            artifact_text,
            json.dumps(result_json or {}, ensure_ascii=False),
            json.dumps(artifact_json or {}, ensure_ascii=False),
        ]
    )
    scores["grounding"], extra, must_cover_hits = score_grounding(case, result_json or {}, combined_text)
    failures.extend(extra)

    scores["decision_fit"], extra = score_decision(case, result_json or {})
    failures.extend(extra)

    scores["chinese_output"], extra = score_chinese(summary_text, artifact_text)
    failures.extend(extra)

    scores["pedagogy"], extra = score_pedagogy(case, artifact_json or {}, artifact_text)
    failures.extend(extra)

    scores["traceability"], extra = score_trace(trace_path)
    failures.extend(extra)

    total = sum(scores.values())
    return {
        "case_id": case["id"],
        "title": case["title"],
        "total": total,
        "scores": scores,
        "must_cover_hits": must_cover_hits,
        "failures": failures,
    }


def render_markdown_report(summary):
    lines = [
        "# Harness Report",
        "",
        f"- generated_at: {summary['generated_at']}",
        f"- run_dir: {summary['run_dir']}",
        f"- average_score: {summary['average_score']}",
        "",
        "| case | score | failures |",
        "| --- | ---: | --- |",
    ]
    for item in summary["cases"]:
        failure_text = ", ".join(item["failures"]) if item["failures"] else "none"
        lines.append(f"| {item['case_id']} | {item['total']} | {failure_text} |")
    return "\n".join(lines) + "\n"


def grade_run(run_dir):
    manifest_path = run_dir / "manifest.json"
    if not manifest_path.exists():
        raise SystemExit(f"manifest.json not found in {run_dir}")
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    cases = load_cases()
    selected = resolve_case_ids(cases, manifest["case_ids"])

    results = []
    for case_id in selected:
        results.append(grade_case(cases[case_id], run_dir / case_id))

    average_score = round(sum(item["total"] for item in results) / max(1, len(results)), 2)
    summary = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "run_dir": str(run_dir),
        "average_score": average_score,
        "cases": results,
    }
    write_json(run_dir / "report.json", summary)
    (run_dir / "REPORT.md").write_text(render_markdown_report(summary), encoding="utf-8")

    print(f"Graded run at {run_dir}")
    print(f"Average score: {average_score}")
    for item in results:
        print(f"  - {item['case_id']}: {item['total']}")


def list_cases():
    cases = load_cases()
    for case in cases.values():
        print(f"{case['id']}\t{case['resource_type']}\t{case['expected_transform']}\t{case['title']}")


def build_parser():
    parser = argparse.ArgumentParser(description="rubickx harness runner")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("list-cases", help="List deterministic harness cases")

    init_parser = subparsers.add_parser("init-run", help="Create a runnable harness workspace")
    init_parser.add_argument("--run-dir", required=True, help="Directory for the generated run")
    init_parser.add_argument("--case", action="append", dest="cases", help="Case id to include")
    init_parser.add_argument("--overwrite", action="store_true", help="Delete run-dir if it exists")

    grade_parser = subparsers.add_parser("grade", help="Grade an existing run directory")
    grade_parser.add_argument("--run-dir", required=True, help="Directory created by init-run")
    return parser


def main():
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "list-cases":
        list_cases()
        return
    if args.command == "init-run":
        init_run(Path(args.run_dir).resolve(), args.cases, args.overwrite)
        return
    if args.command == "grade":
        grade_run(Path(args.run_dir).resolve())
        return
    parser.error(f"Unsupported command: {args.command}")


if __name__ == "__main__":
    main()
