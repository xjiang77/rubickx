import { applyCommandQueue, createRepository, gitEngine, headCommitId } from "./engine";

describe("GitLearningEngine", () => {
  it("parses a semicolon queue without splitting a quoted commit message", () => {
    const parsed = gitEngine.parse('git add app.ts; git commit -m "teach; git"');
    expect(parsed.error).toBeUndefined();
    expect(parsed.actions).toHaveLength(2);
    expect(parsed.actions[1]).toMatchObject({ kind: "commit", message: "teach; git" });
  });

  it("rejects the entire queue when one command is invalid", () => {
    const state = createRepository();
    state.workingTree["app.ts"] = "changed\n";
    const before = structuredClone(state);
    const result = applyCommandQueue(state, "git add app.ts; git explode");
    expect(result.error?.code).toBe("UNKNOWN_COMMAND");
    expect(result.state).toEqual(before);
  });

  it("stages only selected paths", () => {
    const state = createRepository();
    state.workingTree["app.ts"] = "changed app\n";
    state.workingTree["README.md"] = "changed docs\n";
    const result = applyCommandQueue(state, "git add app.ts");
    expect(result.error).toBeUndefined();
    expect(result.state.index["app.ts"]).toBe("changed app\n");
    expect(result.state.index["README.md"]).toBe("# RubickX\n");
    expect(result.trace[0].type).toBe("stage");
  });

  it("does not mutate state for an invalid path", () => {
    const state = createRepository();
    const before = structuredClone(state);
    const result = applyCommandQueue(state, "git add missing.ts");
    expect(result.error?.code).toBe("PATH_NOT_FOUND");
    expect(result.state).toEqual(before);
  });

  it("commits the index snapshot and advances the attached branch", () => {
    const state = createRepository();
    state.workingTree["app.ts"] = "changed\n";
    const result = applyCommandQueue(state, 'git add app.ts; git commit -m "focused change"');
    expect(result.error).toBeUndefined();
    expect(headCommitId(result.state)).toBe("C1");
    expect(result.state.commits.C1.tree["app.ts"]).toBe("changed\n");
    expect(result.state.branches.main).toBe("C1");
  });

  it("resolves a relative ref and detaches HEAD without moving main", () => {
    const state = createRepository();
    state.workingTree["app.ts"] = "v2\n";
    const committed = applyCommandQueue(state, 'git add app.ts; git commit -m "v2"').state;
    const result = applyCommandQueue(committed, "git switch --detach HEAD^");
    expect(result.error).toBeUndefined();
    expect(result.state.head).toEqual({ type: "detached", target: "C0" });
    expect(result.state.branches.main).toBe("C1");
  });

  it("undo restores the previous repository snapshot", () => {
    const state = createRepository();
    state.workingTree["app.ts"] = "changed\n";
    const staged = applyCommandQueue(state, "git add app.ts").state;
    const result = applyCommandQueue(staged, "undo");
    expect(result.error).toBeUndefined();
    expect(result.state.index["app.ts"]).toBe("export const course = 'git';\n");
    expect(result.state.workingTree["app.ts"]).toBe("changed\n");
  });
});
