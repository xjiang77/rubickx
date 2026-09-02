import type { JudgePolicy, JudgeResult } from "../runtime/types";
import { headCommitId } from "./engine";
import type { CommitSnapshot, FileTree, RepositoryCoreState, RepositoryState } from "./types";

function stable(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(stable).join(",")}]`;
  if (value && typeof value === "object") {
    return `{${Object.entries(value as Record<string, unknown>)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, child]) => `${JSON.stringify(key)}:${stable(child)}`)
      .join(",")}}`;
  }
  return JSON.stringify(value);
}

function commitKey(
  commits: Record<string, CommitSnapshot>,
  id: string,
  cache: Map<string, string>,
): string {
  const cached = cache.get(id);
  if (cached) return cached;
  const commit = commits[id];
  if (!commit) return `missing:${id}`;
  const parents = commit.parents.map((parent) => commitKey(commits, parent, cache));
  const key = stable({ parents, tree: commit.tree });
  cache.set(id, key);
  return key;
}

function identity(state: RepositoryCoreState, id: string, hashAgnostic: boolean): string {
  return hashAgnostic ? commitKey(state.commits, id, new Map()) : id;
}

function topology(state: RepositoryCoreState, hashAgnostic: boolean): string {
  const cache = new Map<string, string>();
  const nodes = Object.values(state.commits).map((commit) => ({
    id: hashAgnostic ? commitKey(state.commits, commit.id, cache) : commit.id,
    parents: commit.parents.map((parent) => hashAgnostic ? commitKey(state.commits, parent, cache) : parent),
  }));
  return stable(nodes.sort((left, right) => stable(left).localeCompare(stable(right))));
}

function refs(state: RepositoryCoreState, hashAgnostic: boolean): string {
  const branchRefs = Object.fromEntries(Object.entries(state.branches).map(([name, id]) => [name, identity(state, id, hashAgnostic)]));
  const tagRefs = Object.fromEntries(Object.entries(state.tags).map(([name, id]) => [name, identity(state, id, hashAgnostic)]));
  const head = state.head.type === "branch"
    ? state.head
    : { type: "detached", target: identity(state, state.head.target, hashAgnostic) };
  return stable({ branches: branchRefs, tags: tagRefs, head });
}

function trees(state: RepositoryCoreState, hashAgnostic: boolean): string {
  const entries = Object.values(state.commits).map((commit) => ({
    commit: identity(state, commit.id, hashAgnostic),
    tree: commit.tree,
  }));
  return stable(entries.sort((left, right) => left.commit.localeCompare(right.commit)));
}

function remote(state: RepositoryCoreState, hashAgnostic: boolean): string {
  if (!state.origin) return stable({ origin: null, remoteTracking: {}, tracking: state.tracking });
  const remoteState: RepositoryCoreState = {
    ...state,
    commits: state.origin.commits,
    branches: state.origin.branches,
  };
  return stable({
    branches: Object.fromEntries(Object.entries(state.origin.branches).map(([name, id]) => [name, identity(remoteState, id, hashAgnostic)])),
    remoteTracking: state.remoteTracking,
    tracking: state.tracking,
  });
}

function compareTree(left: FileTree, right: FileTree): boolean {
  return stable(left) === stable(right);
}

export function judgeRepository(
  actual: RepositoryState,
  expected: RepositoryState,
  policy: JudgePolicy,
): JudgeResult {
  const differences: string[] = [];
  for (const facet of policy.compare) {
    if (facet === "topology" && topology(actual, policy.hashAgnostic) !== topology(expected, policy.hashAgnostic)) {
      differences.push("commit topology 还没有达到目标");
    }
    if (facet === "refs" && refs(actual, policy.hashAgnostic) !== refs(expected, policy.hashAgnostic)) {
      differences.push("branch、tag 或 HEAD 指向不符合目标");
    }
    if (facet === "trees" && trees(actual, policy.hashAgnostic) !== trees(expected, policy.hashAgnostic)) {
      differences.push("commit snapshot 不符合目标");
    }
    if (facet === "index" && !compareTree(actual.index, expected.index)) {
      differences.push("index 内容不符合目标");
    }
    if (facet === "workingTree" && !compareTree(actual.workingTree, expected.workingTree)) {
      differences.push("working tree 内容不符合目标");
    }
    if (facet === "remote" && remote(actual, policy.hashAgnostic) !== remote(expected, policy.hashAgnostic)) {
      differences.push("remote state 不符合目标");
    }
  }
  return {
    passed: differences.length === 0,
    summary: differences.length === 0 ? "Repository state 已达到目标。" : "State 还差最后几步。",
    differences,
    score: actual.commandHistory.length,
  };
}

export function describeHead(state: RepositoryState): string {
  return state.head.type === "branch"
    ? `${state.head.target} → ${headCommitId(state)}`
    : `detached → ${state.head.target}`;
}
