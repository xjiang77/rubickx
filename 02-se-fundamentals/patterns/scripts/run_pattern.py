#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
from pathlib import Path


PATTERNS_ROOT = Path(__file__).resolve().parents[1]
REPO_ROOT = PATTERNS_ROOT.parent
CATALOG = json.loads((PATTERNS_ROOT / "catalog.json").read_text(encoding="utf-8"))
JUNIT = PATTERNS_ROOT / ".cache" / "junit-platform-console-standalone-1.10.2.jar"
JACKSON_VERSION = "2.17.2"
JACKSON = [
    PATTERNS_ROOT / ".cache" / f"jackson-annotations-{JACKSON_VERSION}.jar",
    PATTERNS_ROOT / ".cache" / f"jackson-core-{JACKSON_VERSION}.jar",
    PATTERNS_ROOT / ".cache" / f"jackson-databind-{JACKSON_VERSION}.jar",
]


def run(command: list[str], cwd: Path = PATTERNS_ROOT) -> None:
    print("+", " ".join(command), flush=True)
    subprocess.run(command, cwd=cwd, check=True)


def require_tool(name: str) -> None:
    if shutil.which(name) is None:
        raise SystemExit(f"required tool is missing: {name}")


def require_java_dependencies() -> None:
    missing = [path for path in [JUNIT, *JACKSON] if not path.is_file()]
    if missing:
        names = ", ".join(path.name for path in missing)
        raise SystemExit(f"missing Java test dependencies ({names}); run 'make setup'")


def pattern_entries(pattern_id: str | None) -> list[dict[str, object]]:
    if pattern_id is None:
        return CATALOG
    selected = [entry for entry in CATALOG if entry["id"] == pattern_id]
    if not selected:
        raise SystemExit(f"unknown PATTERN id: {pattern_id}")
    return selected


def code_dirs(entries: list[dict[str, object]], language: str) -> list[Path]:
    return [REPO_ROOT / str(entry["path"]) / language for entry in entries]


def test_python(entries: list[dict[str, object]]) -> None:
    print("== Python (pytest) ==", flush=True)
    paths = [str(path) for path in code_dirs(entries, "python")]
    run([os.environ.get("PYTEST", "pytest"), "-q", *paths])


def test_go(entries: list[dict[str, object]], race: bool = False) -> None:
    print(f"== Go ({'race' if race else 'test'}) ==", flush=True)
    packages = [
        "./" + path.relative_to(PATTERNS_ROOT).as_posix()
        for path in code_dirs(entries, "go")
    ]
    command = ["go", "test"]
    if race:
        command.append("-race")
    run([*command, *packages])


def test_java(entries: list[dict[str, object]]) -> None:
    print("== Java (JUnit 5 + Jackson) ==", flush=True)
    require_java_dependencies()
    selected_ids = "-".join(str(entry["order"]) for entry in entries)
    build_dir = PATTERNS_ROOT / ".build" / "java" / selected_ids
    if build_dir.exists():
        shutil.rmtree(build_dir)
    build_dir.mkdir(parents=True)
    sources = sorted(
        source
        for directory in code_dirs(entries, "java")
        for source in directory.glob("*.java")
    )
    sources.extend(sorted((PATTERNS_ROOT / "support" / "java").glob("*.java")))
    classpath = os.pathsep.join(str(path) for path in [JUNIT, *JACKSON])
    run(["javac", "-cp", classpath, "-d", str(build_dir), *map(str, sources)])
    runtime_classpath = os.pathsep.join(str(path) for path in [build_dir, *JACKSON])
    run(
        [
            "java",
            "-jar",
            str(JUNIT),
            "execute",
            "--class-path",
            runtime_classpath,
            "--scan-class-path",
            "--details=summary",
        ]
    )


def test_javascript(entries: list[dict[str, object]]) -> None:
    print("== JavaScript (node:test) ==", flush=True)
    tests = sorted(
        str(test)
        for directory in code_dirs(entries, "js")
        for test in directory.glob("*.test.mjs")
    )
    run(["node", "--test", *tests])


def main() -> int:
    parser = argparse.ArgumentParser(description="Run equivalent tests for Rubickx patterns")
    selection = parser.add_mutually_exclusive_group(required=True)
    selection.add_argument("--pattern")
    selection.add_argument("--all", action="store_true")
    parser.add_argument("--race", action="store_true")
    args = parser.parse_args()

    for tool in (os.environ.get("PYTEST", "pytest"), "go", "javac", "java", "node"):
        require_tool(tool)
    entries = pattern_entries(args.pattern)
    test_python(entries)
    test_go(entries, race=args.race)
    test_java(entries)
    test_javascript(entries)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
