import {
  GIT_PROGRESS_KEY,
  emptyProgress,
  loadProgress,
  recordAttempt,
  saveProgress,
} from "./progress";

describe("progress storage", () => {
  beforeEach(() => localStorage.clear());

  it("persists solved, best, attempts and last lesson", () => {
    const progress = recordAttempt(emptyProgress("three-areas"), "three-areas", 2, true);
    saveProgress(localStorage, progress);
    expect(loadProgress(localStorage, "three-areas")).toEqual(progress);
  });

  it("removes only the Git progress key when JSON is corrupted", () => {
    localStorage.setItem("keep.me", "safe");
    localStorage.setItem(GIT_PROGRESS_KEY, "{broken");
    expect(loadProgress(localStorage, "three-areas")).toEqual(emptyProgress("three-areas"));
    expect(localStorage.getItem(GIT_PROGRESS_KEY)).toBeNull();
    expect(localStorage.getItem("keep.me")).toBe("safe");
  });

  it("resets an unknown schema version", () => {
    localStorage.setItem(GIT_PROGRESS_KEY, JSON.stringify({ schemaVersion: 99, solved: {}, lastLesson: "x" }));
    expect(loadProgress(localStorage, "three-areas")).toEqual(emptyProgress("three-areas"));
  });
});
