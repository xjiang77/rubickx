import { gitCourse, lessons } from "../git/course";
import { courseSchema } from "./schema";

describe("course schema", () => {
  it("validates the released course definition", () => {
    expect(courseSchema.safeParse(gitCourse).success).toBe(true);
  });

  it("requires unique lesson IDs and source mapping", () => {
    expect(new Set(lessons.map((lesson) => lesson.id)).size).toBe(lessons.length);
    for (const lesson of lessons) {
      expect(lesson.sourceRefs.length).toBeGreaterThan(0);
      expect(lesson.demoSteps.length).toBeGreaterThan(0);
      expect(lesson.referenceSolution.length).toBeGreaterThan(0);
      expect(lesson.allowedCommands).toEqual(expect.arrayContaining(["help", "hint", "goal", "reset", "undo", "solution"]));
    }
  });
});
