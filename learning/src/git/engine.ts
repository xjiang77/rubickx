import type {
  EngineError,
  LearningEngine,
  ParseResult,
  SerializableState,
  TerminalOutput,
  TraceEvent,
  Transition,
} from "../runtime/types";
import type {
  CommitSnapshot,
  FileTree,
  GitAction,
  RepositoryCoreState,
  RepositoryState,
} from "./types";

const HELP_TEXT = [
  "Git: status, add <path>, commit -m <message>, log, show [ref], branch [name], switch [--detach] <ref>",
  "Runtime: help, hint, goal, reset, undo, solution",
  "可以省略 git 前缀，也可以用分号顺序执行多条命令。",
].join("\n");

function clone<T>(value: T): T {
  return structuredClone(value);
}

function coreOf(state: RepositoryState): RepositoryCoreState {
  const { undoStack: _undoStack, lastTrace: _lastTrace, ...core } = state;
  return clone(core);
}

export function headCommitId(state: RepositoryCoreState): string {
  return state.head.type === "branch"
    ? state.branches[state.head.target]
    : state.head.target;
}

export function headTree(state: RepositoryCoreState): FileTree {
  return clone(state.commits[headCommitId(state)]?.tree ?? {});
}

export function createRepository(options?: {
  commits?: CommitSnapshot[];
  branches?: Record<string, string>;
  head?: RepositoryCoreState["head"];
  workingTree?: FileTree;
  index?: FileTree;
}): RepositoryState {
  const initialCommit: CommitSnapshot = {
    id: "C0",
    message: "initial snapshot",
    parents: [],
    tree: {
      "README.md": "# RubickX\n",
      "app.ts": "export const course = 'git';\n",
    },
  };
  const commitList = options?.commits ?? [initialCommit];
  const commits = Object.fromEntries(commitList.map((commit) => [commit.id, clone(commit)]));
  const branches = options?.branches ?? { main: commitList.at(-1)?.id ?? "C0" };
  const head = options?.head ?? { type: "branch" as const, target: "main" };
  const currentTree = commits[head.type === "branch" ? branches[head.target] : head.target]?.tree ?? {};
  return {
    commits,
    branches: clone(branches),
    tags: {},
    head: clone(head),
    workingTree: clone(options?.workingTree ?? currentTree),
    index: clone(options?.index ?? currentTree),
    conflicts: {},
    origin: null,
    remoteTracking: {},
    tracking: {},
    commandHistory: [],
    nextCommitNumber: Math.max(0, ...commitList.map((commit) => Number(commit.id.slice(1)) || 0)) + 1,
    undoStack: [],
    lastTrace: [],
  };
}

function splitCommandQueue(input: string): string[] | EngineError {
  const commands: string[] = [];
  let current = "";
  let quote: "'" | '"' | null = null;
  for (const character of input.trim()) {
    if ((character === "'" || character === '"')) {
      if (quote === character) quote = null;
      else if (!quote) quote = character;
      current += character;
      continue;
    }
    if (character === ";" && !quote) {
      if (!current.trim()) return { code: "EMPTY_COMMAND", message: "分号之间缺少命令。" };
      commands.push(current.trim());
      current = "";
      continue;
    }
    current += character;
  }
  if (quote) return { code: "UNCLOSED_QUOTE", message: "引号没有闭合。" };
  if (current.trim()) commands.push(current.trim());
  if (commands.length === 0) return { code: "EMPTY_COMMAND", message: "请输入一条命令。" };
  return commands;
}

function tokenize(command: string): string[] | EngineError {
  const tokens: string[] = [];
  let current = "";
  let quote: "'" | '"' | null = null;
  for (const character of command) {
    if (character === "'" || character === '"') {
      if (quote === character) quote = null;
      else if (!quote) quote = character;
      else current += character;
      continue;
    }
    if (/\s/.test(character) && !quote) {
      if (current) {
        tokens.push(current);
        current = "";
      }
      continue;
    }
    current += character;
  }
  if (quote) return { code: "UNCLOSED_QUOTE", message: "引号没有闭合。" };
  if (current) tokens.push(current);
  return tokens;
}

function parseCommand(raw: string): GitAction | EngineError {
  const tokenResult = tokenize(raw);
  if (!Array.isArray(tokenResult)) return tokenResult;
  const tokens = tokenResult[0] === "git" ? tokenResult.slice(1) : tokenResult;
  const [command, ...args] = tokens;
  if (!command) return { code: "EMPTY_COMMAND", message: "请输入一条命令。" };
  if (["help", "hint", "goal", "reset", "undo", "solution"].includes(command)) {
    if (args.length) return { code: "TOO_MANY_ARGUMENTS", message: `${command} 不接受参数。` };
    const runtimeKinds = {
      help: "help",
      hint: "hint",
      goal: "goal",
      reset: "reset-runtime",
      undo: "undo",
      solution: "solution",
    } as const;
    return { kind: runtimeKinds[command as keyof typeof runtimeKinds], raw };
  }
  if (command === "status" || command === "log") {
    if (args.length) return { code: "TOO_MANY_ARGUMENTS", message: `${command} 暂不接受参数。` };
    return { kind: command, raw };
  }
  if (command === "add") {
    if (!args.length) return { code: "MISSING_PATH", message: "add 需要至少一个 path。" };
    return { kind: "add", paths: args, raw };
  }
  if (command === "commit") {
    const messageFlag = args.indexOf("-m");
    if (messageFlag < 0 || messageFlag !== args.length - 2 || !args[messageFlag + 1]) {
      return { code: "COMMIT_MESSAGE_REQUIRED", message: "请使用 commit -m \"message\"。" };
    }
    return { kind: "commit", message: args[messageFlag + 1], raw };
  }
  if (command === "show") {
    if (args.length > 1) return { code: "TOO_MANY_ARGUMENTS", message: "show 只接受一个 ref。" };
    return { kind: "show", ref: args[0] ?? "HEAD", raw };
  }
  if (command === "branch") {
    if (args.length > 1) return { code: "TOO_MANY_ARGUMENTS", message: "branch 只接受一个 branch name。" };
    return { kind: "branch", name: args[0], raw };
  }
  if (command === "switch") {
    const detach = args[0] === "--detach";
    const ref = detach ? args[1] : args[0];
    if (!ref || args.length !== (detach ? 2 : 1)) {
      return { code: "SWITCH_REF_REQUIRED", message: "请使用 switch <branch> 或 switch --detach <ref>。" };
    }
    return { kind: "switch", ref, detach, raw };
  }
  return { code: "UNKNOWN_COMMAND", message: `暂不支持 ${command}。输入 help 查看可用命令。` };
}

function resolveRef(state: RepositoryCoreState, expression: string): string | undefined {
  const firstOperator = expression.search(/[~^]/);
  const base = firstOperator < 0 ? expression : expression.slice(0, firstOperator);
  let commitId = base === "HEAD"
    ? headCommitId(state)
    : state.branches[base] ?? state.tags[base] ?? (state.commits[base] ? base : undefined);
  if (!commitId) return undefined;
  if (firstOperator < 0) return commitId;
  const suffix = expression.slice(firstOperator);
  const operatorPattern = /(\^|~)(\d*)/g;
  let consumed = "";
  for (const match of suffix.matchAll(operatorPattern)) {
    consumed += match[0];
    const steps = match[1] === "^" ? Number(match[2] || 1) : Number(match[2] || 1);
    for (let step = 0; step < steps; step += 1) {
      commitId = state.commits[commitId]?.parents[0];
      if (!commitId) return undefined;
    }
  }
  return consumed === suffix ? commitId : undefined;
}

function changedPaths(from: FileTree, to: FileTree): string[] {
  return [...new Set([...Object.keys(from), ...Object.keys(to)])]
    .filter((path) => from[path] !== to[path])
    .sort();
}

function treeEquals(left: FileTree, right: FileTree): boolean {
  return changedPaths(left, right).length === 0;
}

function statusOutput(state: RepositoryCoreState): string {
  const staged = changedPaths(headTree(state), state.index);
  const unstaged = changedPaths(state.index, state.workingTree);
  const headLabel = state.head.type === "branch"
    ? `On branch ${state.head.target}`
    : `HEAD detached at ${state.head.target}`;
  if (!staged.length && !unstaged.length) return `${headLabel}\nnothing to commit, working tree clean`;
  const lines = [headLabel];
  if (staged.length) lines.push("Changes to be committed:", ...staged.map((path) => `  staged: ${path}`));
  if (unstaged.length) lines.push("Changes not staged:", ...unstaged.map((path) => `  modified: ${path}`));
  return lines.join("\n");
}

function success(
  state: RepositoryState,
  output: TerminalOutput[],
  trace: TraceEvent[],
): Transition<RepositoryState> {
  state.lastTrace = trace;
  return { state, output, trace };
}

function failure(state: RepositoryState, code: string, message: string): Transition<RepositoryState> {
  const error = { code, message };
  return {
    state,
    error,
    trace: [],
    output: [{ kind: "error", text: message }],
  };
}

function withMutation(state: RepositoryState): RepositoryState {
  const next = clone(state);
  next.undoStack.push(coreOf(state));
  return next;
}

export class GitLearningEngine implements LearningEngine<RepositoryState, GitAction> {
  parse(input: string): ParseResult<GitAction> {
    const queue = splitCommandQueue(input);
    if (!Array.isArray(queue)) return { actions: [], error: queue };
    const actions: GitAction[] = [];
    for (const raw of queue) {
      const parsed = parseCommand(raw);
      if ("code" in parsed) return { actions: [], error: parsed };
      actions.push(parsed);
    }
    return { actions };
  }

  apply(state: RepositoryState, action: GitAction): Transition<RepositoryState> {
    if (action.kind === "status") {
      return success(state, [{ kind: "stdout", text: statusOutput(state) }], [{ type: "inspect", label: "比较 HEAD、index 与 working tree" }]);
    }
    if (action.kind === "log") {
      const lines: string[] = [];
      let current: string | undefined = headCommitId(state);
      while (current) {
        const commit: CommitSnapshot | undefined = state.commits[current];
        if (!commit) break;
        lines.push(`${commit.id} ${commit.message}`);
        current = commit.parents[0];
      }
      return success(state, [{ kind: "stdout", text: lines.join("\n") }], [{ type: "inspect", label: "沿 first-parent 读取 commit graph" }]);
    }
    if (action.kind === "show") {
      const commitId = resolveRef(state, action.ref);
      if (!commitId) return failure(state, "UNKNOWN_REF", `找不到 ref: ${action.ref}`);
      const commit = state.commits[commitId];
      const files = Object.entries(commit.tree).map(([path, content]) => `${path}\n  ${content.trim()}`).join("\n");
      return success(state, [{ kind: "stdout", text: `${commit.id} ${commit.message}\n${files}` }], [{ type: "inspect", label: `查看 ${commitId} 的 snapshot` }]);
    }
    if (action.kind === "help") {
      return success(state, [{ kind: "system", text: HELP_TEXT }], [{ type: "runtime", label: "显示帮助" }]);
    }
    if (["hint", "goal", "reset-runtime", "solution"].includes(action.kind)) {
      return success(state, [], [{ type: "runtime", label: action.kind }]);
    }
    if (action.kind === "undo") {
      const previous = state.undoStack.at(-1);
      if (!previous) return failure(state, "NOTHING_TO_UNDO", "还没有可以 undo 的 state transition。",
      );
      const restored: RepositoryState = {
        ...clone(previous),
        undoStack: state.undoStack.slice(0, -1),
        lastTrace: [{ type: "runtime", label: "恢复前一个 repository snapshot" }],
      };
      return success(restored, [{ kind: "system", text: "已恢复前一个 repository state。" }], restored.lastTrace);
    }
    if (action.kind === "add") {
      for (const path of action.paths) {
        if (!(path in state.workingTree) && !(path in state.index)) {
          return failure(state, "PATH_NOT_FOUND", `pathspec '${path}' did not match any files`);
        }
      }
      const next = withMutation(state);
      for (const path of action.paths) {
        if (path in next.workingTree) next.index[path] = next.workingTree[path];
        else delete next.index[path];
      }
      next.commandHistory.push(action.raw);
      const trace = action.paths.map((path) => ({
        type: "stage",
        label: `${path}: working tree → index`,
        details: { path },
      }));
      return success(next, [{ kind: "stdout", text: `staged ${action.paths.join(", ")}` }], trace);
    }
    if (action.kind === "commit") {
      const parent = headCommitId(state);
      if (treeEquals(state.commits[parent].tree, state.index)) {
        return failure(state, "NOTHING_TO_COMMIT", "nothing to commit；先用 add 更新 index。" );
      }
      const next = withMutation(state);
      const id = `C${next.nextCommitNumber}`;
      next.nextCommitNumber += 1;
      next.commits[id] = {
        id,
        message: action.message,
        parents: [parent],
        tree: clone(next.index),
      };
      if (next.head.type === "branch") next.branches[next.head.target] = id;
      else next.head = { type: "detached", target: id };
      next.commandHistory.push(action.raw);
      const trace = [{
        type: "commit",
        label: `${id}: 固化 index snapshot`,
        details: { id, parent, message: action.message },
      }];
      return success(next, [{ kind: "stdout", text: `[${next.head.type === "branch" ? next.head.target : "detached HEAD"} ${id}] ${action.message}` }], trace);
    }
    if (action.kind === "branch") {
      if (!action.name) {
        const lines = Object.keys(state.branches).sort().map((name) => `${state.head.type === "branch" && state.head.target === name ? "*" : " "} ${name}`);
        return success(state, [{ kind: "stdout", text: lines.join("\n") }], [{ type: "inspect", label: "列出 branches" }]);
      }
      if (state.branches[action.name]) return failure(state, "BRANCH_EXISTS", `branch '${action.name}' already exists`);
      if (!/^[A-Za-z0-9._/-]+$/.test(action.name)) return failure(state, "INVALID_BRANCH", "branch name 含有不支持的字符。" );
      const next = withMutation(state);
      next.branches[action.name] = headCommitId(next);
      next.commandHistory.push(action.raw);
      const trace = [{ type: "ref", label: `${action.name} → ${headCommitId(next)}` }];
      return success(next, [{ kind: "stdout", text: `created branch ${action.name}` }], trace);
    }
    if (action.kind === "switch") {
      if (!treeEquals(state.index, state.workingTree) || !treeEquals(headTree(state), state.index)) {
        return failure(state, "DIRTY_WORKTREE", "请先处理 working tree 与 index 中的改动。" );
      }
      const commitId = resolveRef(state, action.ref);
      if (!commitId) return failure(state, "UNKNOWN_REF", `找不到 ref: ${action.ref}`);
      if (!action.detach && !state.branches[action.ref]) {
        return failure(state, "BRANCH_REQUIRED", `switch ${action.ref} 需要一个 branch；查看历史请使用 --detach。`);
      }
      const next = withMutation(state);
      next.head = action.detach
        ? { type: "detached", target: commitId }
        : { type: "branch", target: action.ref };
      next.index = clone(next.commits[commitId].tree);
      next.workingTree = clone(next.commits[commitId].tree);
      next.commandHistory.push(action.raw);
      const label = action.detach ? `HEAD detached at ${commitId}` : `Switched to branch '${action.ref}'`;
      const trace = [{ type: "checkout", label, details: { commit: commitId } }];
      return success(next, [{ kind: "stdout", text: label }], trace);
    }
    return failure(state, "UNREACHABLE", "无法执行该 action。" );
  }

  snapshot(state: RepositoryState): SerializableState {
    return coreOf(state) as unknown as SerializableState;
  }
}

export const gitEngine = new GitLearningEngine();

export function applyCommandQueue(
  state: RepositoryState,
  input: string,
): Transition<RepositoryState> {
  const parsed = gitEngine.parse(input);
  if (parsed.error) {
    return {
      state,
      error: parsed.error,
      trace: [],
      output: [{ kind: "error", text: parsed.error.message }],
    };
  }
  let current = state;
  const trace: TraceEvent[] = [];
  const output: TerminalOutput[] = [];
  for (const action of parsed.actions) {
    const transition = gitEngine.apply(current, action);
    if (transition.error) {
      return { ...transition, trace: [...trace, ...transition.trace], output: [...output, ...transition.output] };
    }
    current = transition.state;
    trace.push(...transition.trace);
    output.push(...transition.output);
  }
  return { state: current, trace, output };
}
