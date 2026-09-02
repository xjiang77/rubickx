export type SerializableState = Record<string, unknown>;

export interface ParseResult<A> {
  actions: A[];
  error?: EngineError;
}

export interface TerminalOutput {
  kind: "stdout" | "error" | "system";
  text: string;
}

export interface EngineError {
  code: string;
  message: string;
}

export interface TraceEvent {
  type: string;
  label: string;
  details?: Record<string, string | number | boolean>;
}

export interface Transition<S> {
  state: S;
  trace: TraceEvent[];
  output: TerminalOutput[];
  error?: EngineError;
}

export interface LearningEngine<S, A> {
  parse(input: string): ParseResult<A>;
  apply(state: S, action: A): Transition<S>;
  snapshot(state: S): SerializableState;
}

export interface SourceReference {
  id: string;
  title: string;
  url: string;
  note: string;
}

export interface DemoStep {
  id: string;
  title: string;
  command: string;
  explanation: string;
  focus: string;
}

export interface ExerciseDefinition {
  prompt: string;
  success: string;
  starterCommands: string[];
}

export interface CheckpointDefinition {
  question: string;
  options: string[];
  correctIndex: number;
  explanation: string;
}

export type JudgeFacet =
  | "topology"
  | "refs"
  | "trees"
  | "index"
  | "workingTree"
  | "remote";

export interface JudgePolicy {
  compare: JudgeFacet[];
  hashAgnostic: boolean;
}

export interface LessonGoal {
  summary: string;
  acceptance: string[];
}

export interface LessonDefinition<S = SerializableState> {
  id: string;
  sequenceId: string;
  order: number;
  title: string;
  eyebrow: string;
  estimatedMinutes: number;
  startState: S;
  goal: LessonGoal;
  explanation: string[];
  demoSteps: DemoStep[];
  exercise: ExerciseDefinition;
  checkpoint: CheckpointDefinition;
  referenceSolution: string[];
  hints: string[];
  judgePolicy: JudgePolicy;
  sourceRefs: SourceReference[];
  allowedCommands: string[];
}

export interface SequenceDefinition<S = SerializableState> {
  id: string;
  order: number;
  title: string;
  description: string;
  lessons: LessonDefinition<S>[];
}

export interface CourseDefinition<S = SerializableState> {
  id: string;
  version: number;
  title: string;
  description: string;
  locale: string;
  sequences: SequenceDefinition<S>[];
}

export interface JudgeResult {
  passed: boolean;
  summary: string;
  differences: string[];
  score?: number;
}

export interface LessonProgress {
  solved: boolean;
  best: number | null;
  attempts: number;
}

export interface ProgressRecord {
  schemaVersion: number;
  solved: Record<string, LessonProgress>;
  lastLesson: string;
}
