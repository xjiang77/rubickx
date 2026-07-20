#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import sys
from collections import Counter
from pathlib import Path, PurePosixPath


PATTERNS_ROOT = Path(__file__).resolve().parents[1]
REPO_ROOT = PATTERNS_ROOT.parent
CATALOG_PATH = PATTERNS_ROOT / "catalog.json"
PROGRESS_PATH = PATTERNS_ROOT / "PROGRESS.md"
ENGINEERING_MOC_BASENAME = "MOC - Software Engineering：SWE Route 与工程资产"
REQUIRED_KEYS = {
    "id",
    "family",
    "order",
    "name",
    "slug",
    "path",
    "vault_note",
    "contract",
}
EXPECTED_COUNTS = {
    "design.creational": 5,
    "design.structural": 7,
    "design.behavioral": 11,
    "reliability": 6,
    "data-messaging": 7,
    "concurrency": 6,
}
REQUIRED_NOTE_HEADINGS = (
    "TL;DR",
    "Definition",
    "When to use",
    "Participants and Mechanism",
    "Comparison",
    "Good / Bad Examples",
    "Practice Mapping",
    "Pitfalls",
    "Trade-offs",
    "Evidence Protocol",
    "Links and Rationale",
)
REQUIRED_NOTES_HEADINGS = (
    "Behavior Contract",
    "Language Mapping",
    "Proof Boundary",
    "Run",
)


class VerificationError(Exception):
    pass


def fail(message: str) -> None:
    raise VerificationError(message)


def load_catalog() -> list[dict[str, object]]:
    try:
        catalog = json.loads(CATALOG_PATH.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        fail(f"cannot load {CATALOG_PATH}: {error}")
    if not isinstance(catalog, list):
        fail("catalog root must be a JSON array")
    return catalog


def validate_catalog(catalog: list[dict[str, object]]) -> None:
    if len(catalog) != 42:
        fail(f"catalog must contain exactly 42 entries, got {len(catalog)}")

    ids: list[str] = []
    paths: list[str] = []
    vault_notes: list[str] = []
    orders: list[int] = []
    families: Counter[str] = Counter()

    for index, entry in enumerate(catalog, start=1):
        if not isinstance(entry, dict):
            fail(f"catalog entry {index} must be an object")
        if set(entry) != REQUIRED_KEYS:
            missing = sorted(REQUIRED_KEYS - set(entry))
            extra = sorted(set(entry) - REQUIRED_KEYS)
            fail(f"entry {index} key mismatch: missing={missing}, extra={extra}")

        pattern_id = str(entry["id"])
        family = str(entry["family"])
        order = entry["order"]
        slug = str(entry["slug"])
        path = str(entry["path"])
        contract = str(entry["contract"])
        vault_note = str(entry["vault_note"])

        if not re.fullmatch(r"[a-z0-9]+(?:[.-][a-z0-9]+)*", pattern_id):
            fail(f"invalid pattern id: {pattern_id}")
        if family not in EXPECTED_COUNTS:
            fail(f"unknown family for {pattern_id}: {family}")
        if not isinstance(order, int):
            fail(f"order must be an integer for {pattern_id}")
        if not re.fullmatch(r"[a-z0-9]+(?:-[a-z0-9]+)*", slug):
            fail(f"invalid slug for {pattern_id}: {slug}")

        pure_path = PurePosixPath(path)
        if pure_path.is_absolute() or ".." in pure_path.parts:
            fail(f"unsafe path for {pattern_id}: {path}")
        if not path.startswith("patterns/"):
            fail(f"code path must start with patterns/ for {pattern_id}")
        if contract != f"{path}/fixtures/contract.json":
            fail(f"contract path must be under the pattern path for {pattern_id}")
        if not vault_note.startswith("02_Knowledge/02_Engineering/06 - Patterns/"):
            fail(f"unexpected Vault path for {pattern_id}: {vault_note}")
        if not vault_note.endswith(".md"):
            fail(f"Vault note must be Markdown for {pattern_id}")

        ids.append(pattern_id)
        paths.append(path)
        vault_notes.append(vault_note)
        orders.append(order)
        families[family] += 1

    for label, values in (
        ("id", ids),
        ("path", paths),
        ("vault_note", vault_notes),
        ("order", orders),
    ):
        duplicates = sorted(value for value, count in Counter(values).items() if count > 1)
        if duplicates:
            fail(f"duplicate {label}: {duplicates}")

    if sorted(orders) != list(range(1, 43)):
        fail(f"orders must be contiguous 1..42, got {sorted(orders)}")
    if dict(families) != EXPECTED_COUNTS:
        fail(f"family counts mismatch: expected={EXPECTED_COUNTS}, got={dict(families)}")


def parse_progress() -> dict[str, bool]:
    statuses: dict[str, bool] = {}
    pattern = re.compile(r"^- \[([ xX])\] `([^`]+)`")
    for line in PROGRESS_PATH.read_text(encoding="utf-8").splitlines():
        match = pattern.match(line)
        if not match:
            continue
        pattern_id = match.group(2)
        if pattern_id in statuses:
            fail(f"duplicate progress entry: {pattern_id}")
        statuses[pattern_id] = match.group(1).lower() == "x"
    return statuses


def require_files(pattern_dir: Path, pattern_id: str) -> None:
    required = [
        pattern_dir / "NOTES.md",
        pattern_dir / "fixtures" / "contract.json",
        pattern_dir / "python",
        pattern_dir / "go",
        pattern_dir / "java",
        pattern_dir / "js",
    ]
    missing = [str(path.relative_to(REPO_ROOT)) for path in required if not path.exists()]
    if missing:
        fail(f"missing files for {pattern_id}: {missing}")

    python_files = list((pattern_dir / "python").glob("*.py"))
    if not any("test" not in path.stem for path in python_files):
        fail(f"missing Python implementation for {pattern_id}")
    if not any(path.stem.startswith("test_") or path.stem.endswith("_test") for path in python_files):
        fail(f"missing Python test for {pattern_id}")

    go_files = list((pattern_dir / "go").glob("*.go"))
    if not any(not path.name.endswith("_test.go") for path in go_files):
        fail(f"missing Go implementation for {pattern_id}")
    if not any(path.name.endswith("_test.go") for path in go_files):
        fail(f"missing Go test for {pattern_id}")

    java_files = list((pattern_dir / "java").glob("*.java"))
    if not any(not path.name.endswith("Test.java") for path in java_files):
        fail(f"missing Java implementation for {pattern_id}")
    if not any(path.name.endswith("Test.java") for path in java_files):
        fail(f"missing Java test for {pattern_id}")

    js_files = list((pattern_dir / "js").glob("*.mjs"))
    if not any(not path.name.endswith(".test.mjs") for path in js_files):
        fail(f"missing JavaScript implementation for {pattern_id}")
    if not any(path.name.endswith(".test.mjs") for path in js_files):
        fail(f"missing JavaScript test for {pattern_id}")


def validate_contract(contract_path: Path, pattern_id: str) -> None:
    try:
        contract = json.loads(contract_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        fail(f"invalid contract for {pattern_id}: {error}")
    if contract.get("pattern_id") != pattern_id:
        fail(f"contract pattern_id mismatch for {pattern_id}")
    cases = contract.get("cases")
    if not isinstance(cases, list) or len(cases) < 4:
        fail(f"contract for {pattern_id} must contain at least four cases")
    case_ids: list[str] = []
    categories: set[str] = set()
    for case in cases:
        if not isinstance(case, dict):
            fail(f"contract cases must be objects for {pattern_id}")
        for key in ("id", "category", "input"):
            if key not in case:
                fail(f"contract case missing {key} for {pattern_id}")
        if "expected" not in case and "expected_error" not in case:
            fail(f"contract case needs expected or expected_error for {pattern_id}")
        case_ids.append(str(case["id"]))
        categories.add(str(case["category"]))
    if len(case_ids) != len(set(case_ids)):
        fail(f"duplicate contract case id for {pattern_id}")
    required = {"nominal", "boundary", "failure"}
    if not required.issubset(categories):
        fail(f"contract categories missing {sorted(required - categories)} for {pattern_id}")
    if not ({"lifecycle", "non-interference"} & categories):
        fail(f"contract needs lifecycle or non-interference case for {pattern_id}")


def require_markdown_headings(path: Path, headings: tuple[str, ...], label: str) -> str:
    text = path.read_text(encoding="utf-8")
    for heading in headings:
        if not re.search(rf"^## {re.escape(heading)}\s*$", text, re.MULTILINE):
            fail(f"{label} missing heading '## {heading}': {path}")
    return text


def validate_code(catalog: list[dict[str, object]], only_pattern: str | None = None) -> None:
    ids = {str(entry["id"]) for entry in catalog}
    progress = parse_progress()
    if set(progress) != ids:
        fail(
            "PROGRESS.md IDs must match catalog: "
            f"missing={sorted(ids - set(progress))}, extra={sorted(set(progress) - ids)}"
        )

    selected = [entry for entry in catalog if only_pattern is None or entry["id"] == only_pattern]
    if only_pattern and not selected:
        fail(f"unknown pattern: {only_pattern}")

    for entry in selected:
        pattern_id = str(entry["id"])
        if not progress[pattern_id]:
            fail(f"pattern is not marked complete in PROGRESS.md: {pattern_id}")
        pattern_dir = REPO_ROOT / str(entry["path"])
        require_files(pattern_dir, pattern_id)
        validate_contract(REPO_ROOT / str(entry["contract"]), pattern_id)
        require_markdown_headings(
            pattern_dir / "NOTES.md", REQUIRED_NOTES_HEADINGS, f"NOTES for {pattern_id}"
        )


def parse_frontmatter(text: str) -> dict[str, str]:
    if not text.startswith("---\n"):
        return {}
    end = text.find("\n---\n", 4)
    if end < 0:
        return {}
    result: dict[str, str] = {}
    for line in text[4:end].splitlines():
        match = re.match(r"^([A-Za-z0-9_]+):\s*(.*?)\s*$", line)
        if match:
            result[match.group(1)] = match.group(2).strip('"\'')
    return result


def validate_wikilinks(vault_root: Path, scoped_files: list[Path]) -> None:
    markdown_files = list(vault_root.rglob("*.md"))
    stems = {path.stem for path in markdown_files}
    relative_paths = {
        path.relative_to(vault_root).with_suffix("").as_posix() for path in markdown_files
    }
    missing: list[str] = []
    wikilink = re.compile(r"!?\[\[([^\]]+)\]\]")
    for path in scoped_files:
        text = path.read_text(encoding="utf-8")
        for raw_target in wikilink.findall(text):
            target = raw_target.split("|", 1)[0].split("#", 1)[0].strip()
            if not target:
                continue
            if Path(target).suffix and not target.endswith(".md"):
                continue
            normalized = target[:-3] if target.endswith(".md") else target
            if normalized in relative_paths or PurePosixPath(normalized).name in stems:
                continue
            missing.append(f"{path.relative_to(vault_root)} -> {target}")
    if missing:
        fail("missing scoped wikilinks:\n  " + "\n  ".join(sorted(set(missing))))


def validate_vault(
    catalog: list[dict[str, object]], vault_root: Path, only_pattern: str | None = None
) -> None:
    if not vault_root.is_dir():
        fail(f"Vault root does not exist: {vault_root}")
    pattern_root = vault_root / "02_Knowledge/02_Engineering/06 - Patterns"
    navigation_files = list(pattern_root.rglob("00 - *.md"))
    selected = [entry for entry in catalog if only_pattern is None or entry["id"] == only_pattern]
    if only_pattern and not selected:
        fail(f"unknown pattern: {only_pattern}")
    scoped_files: list[Path] = list(navigation_files) if only_pattern is None else []

    for entry in selected:
        pattern_id = str(entry["id"])
        note_path = vault_root / str(entry["vault_note"])
        if not note_path.is_file():
            fail(f"missing Vault note for {pattern_id}: {note_path}")
        text = require_markdown_headings(note_path, REQUIRED_NOTE_HEADINGS, pattern_id)
        frontmatter = parse_frontmatter(text)
        expected = {
            "type": "note",
            "status": "evergreen",
            "pattern_id": pattern_id,
            "pattern_family": str(entry["family"]),
            "rubickx_path": str(entry["path"]),
            "source_map": "[[Engineering Patterns Source Map]]",
        }
        for key, value in expected.items():
            if frontmatter.get(key) != value:
                fail(
                    f"frontmatter mismatch for {pattern_id}: "
                    f"{key} expected={value!r} got={frontmatter.get(key)!r}"
                )
        scenarios = re.findall(r"^## Scenario [123](?:\b|：|:)", text, re.MULTILINE)
        if len(scenarios) != 3:
            fail(f"{pattern_id} must contain exactly three Scenario headings, got {len(scenarios)}")
        if "```mermaid" not in text:
            fail(f"missing Mermaid mechanism diagram for {pattern_id}")
        if f"[[{ENGINEERING_MOC_BASENAME}" not in text:
            fail(
                f"missing Engineering MOC backlink for {pattern_id}: "
                f"expected [[{ENGINEERING_MOC_BASENAME}"
            )
        github_path = f"https://github.com/xjiang77/rubickx/tree/main/{entry['path']}"
        if github_path not in text:
            fail(f"missing rubickx GitHub path for {pattern_id}")
        basename = note_path.stem
        if not any(f"[[{basename}" in nav.read_text(encoding="utf-8") for nav in navigation_files):
            fail(f"note is not covered by a 00 navigation anchor: {basename}")
        scoped_files.append(note_path)

    source_map = vault_root / "05_Resources/Wiki/synthesis/Engineering Patterns Source Map.md"
    if not source_map.is_file():
        fail(f"missing source map: {source_map}")
    scoped_files.append(source_map)
    validate_wikilinks(vault_root, scoped_files)


def main() -> int:
    parser = argparse.ArgumentParser(description="Verify the Rubickx pattern catalog")
    parser.add_argument("--catalog-only", action="store_true")
    parser.add_argument("--pattern")
    parser.add_argument("--vault-root", type=Path)
    args = parser.parse_args()

    try:
        catalog = load_catalog()
        validate_catalog(catalog)
        if not args.catalog_only:
            validate_code(catalog, args.pattern)
        if args.vault_root:
            validate_vault(catalog, args.vault_root.resolve(), args.pattern)
    except VerificationError as error:
        print(f"verify_catalog: FAIL: {error}", file=sys.stderr)
        return 1

    scope = "catalog"
    if not args.catalog_only:
        scope += "+code"
    if args.vault_root:
        scope += "+vault"
    print(f"verify_catalog: PASS ({scope}, 42 patterns)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
