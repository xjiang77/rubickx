import { z } from "zod";

export const sourceReferenceSchema = z.object({
  id: z.string().min(1),
  title: z.string().min(1),
  url: z.url(),
  note: z.string().min(1),
});

export const demoStepSchema = z.object({
  id: z.string().min(1),
  title: z.string().min(1),
  command: z.string().min(1),
  explanation: z.string().min(1),
  focus: z.string().min(1),
});

export const exerciseSchema = z.object({
  prompt: z.string().min(1),
  success: z.string().min(1),
  starterCommands: z.array(z.string()),
});

export const checkpointSchema = z.object({
  question: z.string().min(1),
  options: z.array(z.string().min(1)).min(2),
  correctIndex: z.number().int().nonnegative(),
  explanation: z.string().min(1),
}).refine((value) => value.correctIndex < value.options.length, {
  message: "correctIndex 必须指向一个存在的选项",
});

export const judgePolicySchema = z.object({
  compare: z.array(z.enum([
    "topology",
    "refs",
    "trees",
    "index",
    "workingTree",
    "remote",
  ])).min(1),
  hashAgnostic: z.boolean(),
});

export const lessonSchema = z.object({
  id: z.string().regex(/^[a-z0-9-]+$/),
  sequenceId: z.string().min(1),
  order: z.number().int().positive(),
  title: z.string().min(1),
  eyebrow: z.string().min(1),
  estimatedMinutes: z.number().int().positive(),
  startState: z.record(z.string(), z.unknown()),
  goal: z.object({
    summary: z.string().min(1),
    acceptance: z.array(z.string().min(1)).min(1),
  }),
  explanation: z.array(z.string().min(1)).min(1),
  demoSteps: z.array(demoStepSchema).min(1),
  exercise: exerciseSchema,
  checkpoint: checkpointSchema,
  referenceSolution: z.array(z.string().min(1)).min(1),
  hints: z.array(z.string().min(1)).min(1),
  judgePolicy: judgePolicySchema,
  sourceRefs: z.array(sourceReferenceSchema).min(1),
  allowedCommands: z.array(z.string().min(1)).min(1),
});

export const sequenceSchema = z.object({
  id: z.string().regex(/^[a-z0-9-]+$/),
  order: z.number().int().positive(),
  title: z.string().min(1),
  description: z.string().min(1),
  lessons: z.array(lessonSchema).min(1),
});

export const courseSchema = z.object({
  id: z.string().regex(/^[a-z0-9-]+$/),
  version: z.number().int().positive(),
  title: z.string().min(1),
  description: z.string().min(1),
  locale: z.string().min(2),
  sequences: z.array(sequenceSchema).min(1),
}).superRefine((course, context) => {
  const lessonIds = new Set<string>();
  for (const sequence of course.sequences) {
    for (const lesson of sequence.lessons) {
      if (lesson.sequenceId !== sequence.id) {
        context.addIssue({
          code: "custom",
          path: ["sequences", sequence.order - 1, "lessons", lesson.order - 1, "sequenceId"],
          message: "lesson.sequenceId 必须匹配所在 sequence",
        });
      }
      if (lessonIds.has(lesson.id)) {
        context.addIssue({
          code: "custom",
          path: ["sequences", sequence.order - 1, "lessons", lesson.order - 1, "id"],
          message: `重复 lesson id: ${lesson.id}`,
        });
      }
      lessonIds.add(lesson.id);
    }
  }
});
