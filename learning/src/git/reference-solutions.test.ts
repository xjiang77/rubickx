import { gitCourse, lessons, targetStateForLesson } from "./course";
import { applyCommandQueue } from "./engine";
import { judgeRepository } from "./judge";

describe("Fundamentals reference solutions", () => {
  it("ships exactly three complete Gate 1 lessons", () => {
    expect(gitCourse.sequences).toHaveLength(1);
    expect(lessons.map((lesson) => lesson.id)).toEqual([
      "three-areas",
      "stage-and-commit",
      "commit-graph-head",
    ]);
  });

  it.each(lessons.map((lesson) => [lesson.id, lesson] as const))(
    "%s reaches the declared target state",
    (_id, lesson) => {
      let state = structuredClone(lesson.startState);
      for (const command of lesson.referenceSolution) {
        const transition = applyCommandQueue(state, command);
        expect(transition.error).toBeUndefined();
        state = transition.state;
      }
      expect(judgeRepository(state, targetStateForLesson(lesson), lesson.judgePolicy)).toMatchObject({ passed: true });
    },
  );

  it("accepts equivalent commit messages because the judge compares state", () => {
    const lesson = lessons[1];
    let state = structuredClone(lesson.startState);
    const alternative = [
      "git add app.ts",
      'git commit -m "code"',
      "git add README.md",
      'git commit -m "docs"',
    ];
    for (const command of alternative) state = applyCommandQueue(state, command).state;
    expect(judgeRepository(state, targetStateForLesson(lesson), lesson.judgePolicy).passed).toBe(true);
  });

  it("rejects a single commit when the goal requires two snapshots", () => {
    const lesson = lessons[1];
    const result = applyCommandQueue(
      structuredClone(lesson.startState),
      'git add app.ts README.md; git commit -m "squashed"',
    );
    const judged = judgeRepository(result.state, targetStateForLesson(lesson), lesson.judgePolicy);
    expect(judged.passed).toBe(false);
    expect(judged.differences).toContain("commit topology 还没有达到目标");
  });
});
