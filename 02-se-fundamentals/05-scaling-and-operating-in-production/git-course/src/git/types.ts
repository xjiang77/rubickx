import type { TraceEvent } from "../runtime/types";

export type FileTree = Record<string, string>;

export interface CommitSnapshot {
  id: string;
  message: string;
  parents: string[];
  tree: FileTree;
}

export type HeadState =
  | { type: "branch"; target: string }
  | { type: "detached"; target: string };

export interface RepositoryCoreState {
  commits: Record<string, CommitSnapshot>;
  branches: Record<string, string>;
  tags: Record<string, string>;
  head: HeadState;
  workingTree: FileTree;
  index: FileTree;
  conflicts: Record<string, { ours: string; theirs: string }>;
  origin: null | {
    commits: Record<string, CommitSnapshot>;
    branches: Record<string, string>;
  };
  remoteTracking: Record<string, string>;
  tracking: Record<string, string>;
  commandHistory: string[];
  nextCommitNumber: number;
}

export interface RepositoryState extends RepositoryCoreState {
  undoStack: RepositoryCoreState[];
  lastTrace: TraceEvent[];
}

export type GitAction =
  | { kind: "status"; raw: string }
  | { kind: "add"; paths: string[]; raw: string }
  | { kind: "commit"; message: string; raw: string }
  | { kind: "log"; raw: string }
  | { kind: "show"; ref: string; raw: string }
  | { kind: "branch"; name?: string; raw: string }
  | { kind: "switch"; ref: string; detach: boolean; raw: string }
  | { kind: "help"; raw: string }
  | { kind: "hint"; raw: string }
  | { kind: "goal"; raw: string }
  | { kind: "reset-runtime"; raw: string }
  | { kind: "undo"; raw: string }
  | { kind: "solution"; raw: string };
