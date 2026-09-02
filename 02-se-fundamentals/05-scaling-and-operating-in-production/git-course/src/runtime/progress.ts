import type { LessonProgress, ProgressRecord } from "./types";

export const GIT_PROGRESS_KEY = "rubickx.learning.git.v1";
export const PROGRESS_SCHEMA_VERSION = 1;

export function emptyProgress(firstLesson: string): ProgressRecord {
  return {
    schemaVersion: PROGRESS_SCHEMA_VERSION,
    solved: {},
    lastLesson: firstLesson,
  };
}

function isLessonProgress(value: unknown): value is LessonProgress {
  if (!value || typeof value !== "object") return false;
  const progress = value as Partial<LessonProgress>;
  return typeof progress.solved === "boolean"
    && (progress.best === null || typeof progress.best === "number")
    && typeof progress.attempts === "number";
}

function isProgressRecord(value: unknown): value is ProgressRecord {
  if (!value || typeof value !== "object") return false;
  const record = value as Partial<ProgressRecord>;
  return record.schemaVersion === PROGRESS_SCHEMA_VERSION
    && typeof record.lastLesson === "string"
    && Boolean(record.solved)
    && typeof record.solved === "object"
    && Object.values(record.solved).every(isLessonProgress);
}

export function loadProgress(storage: Storage, firstLesson: string): ProgressRecord {
  const raw = storage.getItem(GIT_PROGRESS_KEY);
  if (!raw) return emptyProgress(firstLesson);
  try {
    const parsed: unknown = JSON.parse(raw);
    if (isProgressRecord(parsed)) return parsed;
  } catch {
    storage.removeItem(GIT_PROGRESS_KEY);
    return emptyProgress(firstLesson);
  }
  storage.removeItem(GIT_PROGRESS_KEY);
  return emptyProgress(firstLesson);
}

export function saveProgress(storage: Storage, progress: ProgressRecord): void {
  storage.setItem(GIT_PROGRESS_KEY, JSON.stringify(progress));
}

export function recordAttempt(
  progress: ProgressRecord,
  lessonId: string,
  commandCount: number,
  solved: boolean,
): ProgressRecord {
  const previous = progress.solved[lessonId] ?? { solved: false, best: null, attempts: 0 };
  const nextBest = solved
    ? previous.best === null ? commandCount : Math.min(previous.best, commandCount)
    : previous.best;
  return {
    ...progress,
    lastLesson: lessonId,
    solved: {
      ...progress.solved,
      [lessonId]: {
        solved: previous.solved || solved,
        best: nextBest,
        attempts: previous.attempts + 1,
      },
    },
  };
}
