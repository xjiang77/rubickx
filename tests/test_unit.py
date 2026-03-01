#!/usr/bin/env python3
"""
test_unit.py - Offline unit tests for agent modules.

Tests pure functions and classes that require no API keys:
  - TodoManager (s03): validation, rendering
  - SkillLoader (s05): frontmatter parsing
  - estimate_tokens (s06): token estimation
  - EventBus (s12): JSONL append + list_recent
  - TaskManager (s12): CRUD, worktree binding
  - WorktreeManager (s12): name validation

Usage:
    python tests/test_unit.py
"""

import json
import os
import re
import sys
import tempfile
import time
from pathlib import Path

# ---------------------------------------------------------------------------
# Helpers: inline the pure functions so we don't import modules that call
# Anthropic() at module level.
# ---------------------------------------------------------------------------


# -- TodoManager (from s03) -------------------------------------------------

class TodoManager:
    def __init__(self):
        self.items = []

    def update(self, items: list) -> str:
        if len(items) > 20:
            raise ValueError("Max 20 todos allowed")
        validated = []
        in_progress_count = 0
        for i, item in enumerate(items):
            text = str(item.get("text", "")).strip()
            status = str(item.get("status", "pending")).lower()
            item_id = str(item.get("id", str(i + 1)))
            if not text:
                raise ValueError(f"Item {item_id}: text required")
            if status not in ("pending", "in_progress", "completed"):
                raise ValueError(f"Item {item_id}: invalid status '{status}'")
            if status == "in_progress":
                in_progress_count += 1
            validated.append({"id": item_id, "text": text, "status": status})
        if in_progress_count > 1:
            raise ValueError("Only one task can be in_progress at a time")
        self.items = validated
        return self.render()

    def render(self) -> str:
        if not self.items:
            return "No todos."
        lines = []
        for item in self.items:
            marker = {"pending": "[ ]", "in_progress": "[>]", "completed": "[x]"}[item["status"]]
            lines.append(f"{marker} #{item['id']}: {item['text']}")
        done = sum(1 for t in self.items if t["status"] == "completed")
        lines.append(f"\n({done}/{len(self.items)} completed)")
        return "\n".join(lines)


# -- SkillLoader frontmatter parser (from s05) -------------------------------

def parse_frontmatter(text: str) -> tuple:
    match = re.match(r"^---\n(.*?)\n---\n(.*)", text, re.DOTALL)
    if not match:
        return {}, text
    meta = {}
    for line in match.group(1).strip().splitlines():
        if ":" in line:
            key, val = line.split(":", 1)
            meta[key.strip()] = val.strip()
    return meta, match.group(2).strip()


# -- estimate_tokens (from s06) ----------------------------------------------

def estimate_tokens(messages: list) -> int:
    return len(str(messages)) // 4


# -- EventBus (from s12) ----------------------------------------------------

class EventBus:
    def __init__(self, event_log_path: Path):
        self.path = event_log_path
        self.path.parent.mkdir(parents=True, exist_ok=True)
        if not self.path.exists():
            self.path.write_text("")

    def emit(self, event: str, task=None, worktree=None, error=None):
        payload = {
            "event": event,
            "ts": time.time(),
            "task": task or {},
            "worktree": worktree or {},
        }
        if error:
            payload["error"] = error
        with self.path.open("a", encoding="utf-8") as f:
            f.write(json.dumps(payload) + "\n")

    def list_recent(self, limit: int = 20) -> str:
        n = max(1, min(int(limit or 20), 200))
        lines = self.path.read_text(encoding="utf-8").splitlines()
        recent = lines[-n:]
        items = []
        for line in recent:
            try:
                items.append(json.loads(line))
            except Exception:
                items.append({"event": "parse_error", "raw": line})
        return json.dumps(items, indent=2)


# -- TaskManager (from s12) -------------------------------------------------

class TaskManager:
    def __init__(self, tasks_dir: Path):
        self.dir = tasks_dir
        self.dir.mkdir(parents=True, exist_ok=True)
        self._next_id = self._max_id() + 1

    def _max_id(self) -> int:
        ids = []
        for f in self.dir.glob("task_*.json"):
            try:
                ids.append(int(f.stem.split("_")[1]))
            except Exception:
                pass
        return max(ids) if ids else 0

    def _path(self, task_id: int) -> Path:
        return self.dir / f"task_{task_id}.json"

    def _load(self, task_id: int) -> dict:
        path = self._path(task_id)
        if not path.exists():
            raise ValueError(f"Task {task_id} not found")
        return json.loads(path.read_text())

    def _save(self, task: dict):
        self._path(task["id"]).write_text(json.dumps(task, indent=2))

    def create(self, subject: str, description: str = "") -> str:
        task = {
            "id": self._next_id,
            "subject": subject,
            "description": description,
            "status": "pending",
            "owner": "",
            "worktree": "",
            "blockedBy": [],
            "created_at": time.time(),
            "updated_at": time.time(),
        }
        self._save(task)
        self._next_id += 1
        return json.dumps(task, indent=2)

    def get(self, task_id: int) -> str:
        return json.dumps(self._load(task_id), indent=2)

    def exists(self, task_id: int) -> bool:
        return self._path(task_id).exists()

    def update(self, task_id: int, status=None, owner=None) -> str:
        task = self._load(task_id)
        if status:
            if status not in ("pending", "in_progress", "completed"):
                raise ValueError(f"Invalid status: {status}")
            task["status"] = status
        if owner is not None:
            task["owner"] = owner
        task["updated_at"] = time.time()
        self._save(task)
        return json.dumps(task, indent=2)

    def bind_worktree(self, task_id: int, worktree: str, owner: str = "") -> str:
        task = self._load(task_id)
        task["worktree"] = worktree
        if owner:
            task["owner"] = owner
        if task["status"] == "pending":
            task["status"] = "in_progress"
        task["updated_at"] = time.time()
        self._save(task)
        return json.dumps(task, indent=2)

    def unbind_worktree(self, task_id: int) -> str:
        task = self._load(task_id)
        task["worktree"] = ""
        task["updated_at"] = time.time()
        self._save(task)
        return json.dumps(task, indent=2)

    def list_all(self) -> str:
        tasks = []
        for f in sorted(self.dir.glob("task_*.json")):
            tasks.append(json.loads(f.read_text()))
        if not tasks:
            return "No tasks."
        lines = []
        for t in tasks:
            marker = {"pending": "[ ]", "in_progress": "[>]", "completed": "[x]"}.get(
                t["status"], "[?]"
            )
            owner = f" owner={t['owner']}" if t.get("owner") else ""
            wt = f" wt={t['worktree']}" if t.get("worktree") else ""
            lines.append(f"{marker} #{t['id']}: {t['subject']}{owner}{wt}")
        return "\n".join(lines)


# -- Worktree name validation (from s12) ------------------------------------

_NAME_RE = re.compile(r"[A-Za-z0-9._-]{1,40}")

def validate_worktree_name(name: str):
    if not _NAME_RE.fullmatch(name or ""):
        raise ValueError(
            "Invalid worktree name. Use 1-40 chars: letters, numbers, ., _, -"
        )


# ===========================================================================
# Tests
# ===========================================================================

passed = 0
failed = 0
errors = []


def run_test(name, fn):
    global passed, failed
    try:
        fn()
        passed += 1
        print(f"  PASS  {name}")
    except Exception as e:
        failed += 1
        errors.append((name, e))
        print(f"  FAIL  {name}: {e}")


# -- TodoManager tests -------------------------------------------------------

def test_todo_empty():
    t = TodoManager()
    assert t.render() == "No todos."

def test_todo_create():
    t = TodoManager()
    result = t.update([{"text": "Buy milk", "status": "pending"}])
    assert "[ ] #1: Buy milk" in result
    assert "(0/1 completed)" in result

def test_todo_mixed_statuses():
    t = TodoManager()
    result = t.update([
        {"text": "Done", "status": "completed"},
        {"text": "Doing", "status": "in_progress"},
        {"text": "Later", "status": "pending"},
    ])
    assert "[x]" in result
    assert "[>]" in result
    assert "[ ]" in result
    assert "(1/3 completed)" in result

def test_todo_invalid_status():
    t = TodoManager()
    try:
        t.update([{"text": "Bad", "status": "bogus"}])
        assert False, "Should have raised ValueError"
    except ValueError as e:
        assert "invalid status" in str(e)

def test_todo_empty_text():
    t = TodoManager()
    try:
        t.update([{"text": "", "status": "pending"}])
        assert False, "Should have raised ValueError"
    except ValueError as e:
        assert "text required" in str(e)

def test_todo_multiple_in_progress():
    t = TodoManager()
    try:
        t.update([
            {"text": "A", "status": "in_progress"},
            {"text": "B", "status": "in_progress"},
        ])
        assert False, "Should have raised ValueError"
    except ValueError as e:
        assert "Only one" in str(e)

def test_todo_max_limit():
    t = TodoManager()
    try:
        t.update([{"text": f"T{i}", "status": "pending"} for i in range(21)])
        assert False, "Should have raised ValueError"
    except ValueError as e:
        assert "Max 20" in str(e)


# -- Frontmatter parser tests ------------------------------------------------

def test_frontmatter_valid():
    text = "---\ntitle: My Skill\ndescription: Does things\n---\nBody content here."
    meta, body = parse_frontmatter(text)
    assert meta["title"] == "My Skill"
    assert meta["description"] == "Does things"
    assert body == "Body content here."

def test_frontmatter_missing():
    text = "Just plain text."
    meta, body = parse_frontmatter(text)
    assert meta == {}
    assert body == "Just plain text."

def test_frontmatter_colon_in_value():
    text = "---\nurl: http://example.com\n---\nBody"
    meta, body = parse_frontmatter(text)
    assert meta["url"] == "http://example.com"


# -- estimate_tokens tests ---------------------------------------------------

def test_estimate_tokens_empty():
    assert estimate_tokens([]) == 0

def test_estimate_tokens_content():
    msgs = [{"role": "user", "content": "Hello world"}]
    tokens = estimate_tokens(msgs)
    assert tokens > 0
    assert isinstance(tokens, int)


# -- EventBus tests -----------------------------------------------------------

def test_eventbus_emit_and_list():
    with tempfile.TemporaryDirectory() as d:
        bus = EventBus(Path(d) / "events.jsonl")
        bus.emit("test.create", task={"id": 1}, worktree={"name": "wt1"})
        bus.emit("test.remove", task={"id": 1}, error="something went wrong")
        recent = bus.list_recent(10)
        items = json.loads(recent)
        assert len(items) == 2
        assert items[0]["event"] == "test.create"
        assert items[1]["event"] == "test.remove"
        assert items[1]["error"] == "something went wrong"

def test_eventbus_limit():
    with tempfile.TemporaryDirectory() as d:
        bus = EventBus(Path(d) / "events.jsonl")
        for i in range(10):
            bus.emit(f"evt.{i}")
        items = json.loads(bus.list_recent(3))
        assert len(items) == 3
        assert items[0]["event"] == "evt.7"

def test_eventbus_empty():
    with tempfile.TemporaryDirectory() as d:
        bus = EventBus(Path(d) / "events.jsonl")
        items = json.loads(bus.list_recent(5))
        assert items == []


# -- TaskManager tests --------------------------------------------------------

def test_task_create():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        result = json.loads(tm.create("Fix bug"))
        assert result["id"] == 1
        assert result["status"] == "pending"
        assert result["subject"] == "Fix bug"

def test_task_get():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        tm.create("Task A")
        task = json.loads(tm.get(1))
        assert task["subject"] == "Task A"

def test_task_get_nonexistent():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        try:
            tm.get(999)
            assert False, "Should have raised ValueError"
        except ValueError as e:
            assert "not found" in str(e)

def test_task_update_status():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        tm.create("Task A")
        result = json.loads(tm.update(1, status="in_progress"))
        assert result["status"] == "in_progress"

def test_task_update_invalid_status():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        tm.create("Task A")
        try:
            tm.update(1, status="bogus")
            assert False, "Should have raised ValueError"
        except ValueError as e:
            assert "Invalid status" in str(e)

def test_task_bind_worktree():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        tm.create("Task A")
        result = json.loads(tm.bind_worktree(1, "my-wt", "alice"))
        assert result["worktree"] == "my-wt"
        assert result["owner"] == "alice"
        assert result["status"] == "in_progress"  # auto-promoted from pending

def test_task_unbind_worktree():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        tm.create("Task A")
        tm.bind_worktree(1, "my-wt")
        result = json.loads(tm.unbind_worktree(1))
        assert result["worktree"] == ""

def test_task_list_all():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        tm.create("Alpha")
        tm.create("Beta")
        tm.update(2, status="completed")
        listing = tm.list_all()
        assert "[ ] #1: Alpha" in listing
        assert "[x] #2: Beta" in listing

def test_task_id_increment():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        j1 = json.loads(tm.create("First"))
        j2 = json.loads(tm.create("Second"))
        assert j1["id"] == 1
        assert j2["id"] == 2

def test_task_exists():
    with tempfile.TemporaryDirectory() as d:
        tm = TaskManager(Path(d))
        tm.create("Task A")
        assert tm.exists(1) is True
        assert tm.exists(99) is False

def test_task_persistence():
    with tempfile.TemporaryDirectory() as d:
        tm1 = TaskManager(Path(d))
        tm1.create("Persistent")
        # New instance should pick up existing tasks
        tm2 = TaskManager(Path(d))
        task = json.loads(tm2.get(1))
        assert task["subject"] == "Persistent"
        # next_id should continue from max
        j = json.loads(tm2.create("Second"))
        assert j["id"] == 2


# -- Worktree name validation tests ------------------------------------------

def test_wt_name_valid():
    for name in ["auth-refactor", "test.branch", "feature_1", "A", "a" * 40]:
        validate_worktree_name(name)  # should not raise

def test_wt_name_empty():
    try:
        validate_worktree_name("")
        assert False
    except ValueError:
        pass

def test_wt_name_too_long():
    try:
        validate_worktree_name("a" * 41)
        assert False
    except ValueError:
        pass

def test_wt_name_special_chars():
    for bad in ["hello world", "path/inject", "name@host", "rm -rf", "wt;echo"]:
        try:
            validate_worktree_name(bad)
            assert False, f"Should reject: {bad}"
        except ValueError:
            pass


# ===========================================================================
# Runner
# ===========================================================================

if __name__ == "__main__":
    print("=== Agent Unit Tests ===\n")

    all_tests = [
        # TodoManager
        ("TodoManager: empty render", test_todo_empty),
        ("TodoManager: create todo", test_todo_create),
        ("TodoManager: mixed statuses", test_todo_mixed_statuses),
        ("TodoManager: invalid status", test_todo_invalid_status),
        ("TodoManager: empty text", test_todo_empty_text),
        ("TodoManager: multiple in_progress", test_todo_multiple_in_progress),
        ("TodoManager: max limit", test_todo_max_limit),
        # Frontmatter parser
        ("Frontmatter: valid parse", test_frontmatter_valid),
        ("Frontmatter: missing", test_frontmatter_missing),
        ("Frontmatter: colon in value", test_frontmatter_colon_in_value),
        # estimate_tokens
        ("estimate_tokens: empty", test_estimate_tokens_empty),
        ("estimate_tokens: content", test_estimate_tokens_content),
        # EventBus
        ("EventBus: emit and list", test_eventbus_emit_and_list),
        ("EventBus: limit", test_eventbus_limit),
        ("EventBus: empty log", test_eventbus_empty),
        # TaskManager
        ("TaskManager: create", test_task_create),
        ("TaskManager: get", test_task_get),
        ("TaskManager: get nonexistent", test_task_get_nonexistent),
        ("TaskManager: update status", test_task_update_status),
        ("TaskManager: invalid status", test_task_update_invalid_status),
        ("TaskManager: bind worktree", test_task_bind_worktree),
        ("TaskManager: unbind worktree", test_task_unbind_worktree),
        ("TaskManager: list all", test_task_list_all),
        ("TaskManager: id increment", test_task_id_increment),
        ("TaskManager: exists", test_task_exists),
        ("TaskManager: persistence", test_task_persistence),
        # Worktree name validation
        ("Worktree name: valid names", test_wt_name_valid),
        ("Worktree name: empty", test_wt_name_empty),
        ("Worktree name: too long", test_wt_name_too_long),
        ("Worktree name: special chars", test_wt_name_special_chars),
    ]

    for name, fn in all_tests:
        run_test(name, fn)

    print(f"\n=== Results: {passed}/{passed + failed} passed ===")
    if errors:
        print("\nFailures:")
        for name, e in errors:
            print(f"  {name}: {e}")
    sys.exit(0 if failed == 0 else 1)
