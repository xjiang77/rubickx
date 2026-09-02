import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { gitCourse, lessons, targetStateForLesson } from "../git/course";
import { gitEngine } from "../git/engine";
import { judgeRepository } from "../git/judge";
import type { GitAction, RepositoryState } from "../git/types";
import {
  emptyProgress,
  loadProgress,
  recordAttempt,
  saveProgress,
} from "../runtime/progress";
import type { JudgeResult, LessonDefinition, ProgressRecord, TerminalOutput } from "../runtime/types";
import { RepositoryView } from "./RepositoryView";

interface TerminalLine extends TerminalOutput {
  id: number;
}

const plannedSequences = [
  { title: "Sequence 2 · Branching", lessons: "Branch / Merge / Conflict", gate: "Gate 2" },
  { title: "Sequence 3 · History Repair", lessons: "Rebase / Reset / Revert / Cherry-pick", gate: "Gate 3" },
  { title: "Sequence 4 · Remote Collaboration", lessons: "Fetch / Pull / Push safety", gate: "Gate 4" },
];

function cloneState(state: RepositoryState): RepositoryState {
  return structuredClone(state);
}

function actionCommand(action: GitAction): string {
  const mapping: Record<GitAction["kind"], string> = {
    status: "status",
    add: "add",
    commit: "commit",
    log: "log",
    show: "show",
    branch: "branch",
    switch: "switch",
    help: "help",
    hint: "hint",
    goal: "goal",
    "reset-runtime": "reset",
    undo: "undo",
    solution: "solution",
  };
  return mapping[action.kind];
}

function phaseLabel(index: number, active: number): string {
  if (index < active) return "完成";
  if (index === active) return "当前";
  return "等待";
}

function delayForMotion(): Promise<void> {
  if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) return Promise.resolve();
  return new Promise((resolve) => window.setTimeout(resolve, 180));
}

function LessonNavigation({
  selectedId,
  progress,
  onSelect,
}: {
  selectedId: string;
  progress: ProgressRecord;
  onSelect: (lessonId: string) => void;
}) {
  const solvedCount = Object.values(progress.solved).filter((item) => item.solved).length;
  return (
    <aside className="course-sidebar">
      <a className="brand" href="../">
        <span className="brand-mark">RX</span>
        <span><strong>RubickX</strong><small>Learning Runtime</small></span>
      </a>
      <div className="sequence-summary">
        <span>Gate 1 · Fundamentals</span>
        <strong>{solvedCount} / {lessons.length} lessons</strong>
        <div className="progress-track"><span style={{ width: `${(solvedCount / lessons.length) * 100}%` }} /></div>
      </div>
      <nav aria-label="课程章节">
        <p className="nav-label">当前 Sequence</p>
        {lessons.map((lesson, index) => {
          const solved = progress.solved[lesson.id]?.solved;
          return (
            <button
              key={lesson.id}
              className={`lesson-nav ${lesson.id === selectedId ? "lesson-nav-active" : ""}`}
              onClick={() => onSelect(lesson.id)}
            >
              <span className="lesson-number">{solved ? "✓" : String(index + 1).padStart(2, "0")}</span>
              <span><strong>{lesson.title}</strong><small>{lesson.estimatedMinutes} min</small></span>
            </button>
          );
        })}
        <p className="nav-label nav-label-roadmap">Roadmap</p>
        {plannedSequences.map((sequence) => (
          <div className="planned-sequence" key={sequence.gate}>
            <span>⌁</span>
            <div><strong>{sequence.title}</strong><small>{sequence.lessons}</small></div>
          </div>
        ))}
      </nav>
      <div className="sidebar-note">Pure browser simulator<br />No real Git · No network</div>
    </aside>
  );
}

function GuidedDemo({ lesson }: { lesson: LessonDefinition<RepositoryState> }) {
  const [step, setStep] = useState(0);
  const [state, setState] = useState(() => cloneState(lesson.startState));
  const [output, setOutput] = useState("点击 Next transition，观察 command 如何改变 state。 ");

  useEffect(() => {
    setStep(0);
    setState(cloneState(lesson.startState));
    setOutput("点击 Next transition，观察 command 如何改变 state。 ");
  }, [lesson]);

  const runNext = () => {
    const demo = lesson.demoSteps[step];
    if (!demo) return;
    const parsed = gitEngine.parse(demo.command);
    if (parsed.error) {
      setOutput(parsed.error.message);
      return;
    }
    let next = state;
    const lines: string[] = [];
    for (const action of parsed.actions) {
      const transition = gitEngine.apply(next, action);
      if (transition.error) {
        lines.push(transition.error.message);
        break;
      }
      next = transition.state;
      lines.push(...transition.output.map((item) => item.text));
    }
    setState(next);
    setOutput(lines.join("\n") || demo.explanation);
    setStep((current) => current + 1);
  };

  const reset = () => {
    setStep(0);
    setState(cloneState(lesson.startState));
    setOutput("Demo 已重置。 ");
  };

  const activeStep = lesson.demoSteps[Math.min(step, lesson.demoSteps.length - 1)];
  return (
    <section className="lesson-section demo-section" aria-labelledby="demo-title">
      <div className="section-heading">
        <span className="step-chip">02</span>
        <div><span className="section-kicker">Guided demo</span><h2 id="demo-title">逐个观察 transition</h2></div>
        <span className="step-count">{Math.min(step + 1, lesson.demoSteps.length)} / {lesson.demoSteps.length}</span>
      </div>
      <div className="demo-layout">
        <div className="demo-copy">
          <span className="focus-label">Focus · {activeStep.focus}</span>
          <h3>{activeStep.title}</h3>
          <code className="command-card">$ {activeStep.command}</code>
          <p>{activeStep.explanation}</p>
          <div className="demo-controls">
            <button className="button button-primary" onClick={runNext} disabled={step >= lesson.demoSteps.length}>
              {step >= lesson.demoSteps.length ? "Demo complete" : "Next transition"}
            </button>
            <button className="button button-ghost" onClick={reset}>Reset demo</button>
          </div>
          <pre className="demo-output" aria-live="polite">{output}</pre>
        </div>
        <RepositoryView state={state} compact />
      </div>
    </section>
  );
}

function PracticeTerminal({
  lesson,
  state,
  setState,
  lines,
  setLines,
  commandCount,
  setCommandCount,
  onReset,
  onBusyChange,
}: {
  lesson: LessonDefinition<RepositoryState>;
  state: RepositoryState;
  setState: (state: RepositoryState) => void;
  lines: TerminalLine[];
  setLines: (updater: (lines: TerminalLine[]) => TerminalLine[]) => void;
  commandCount: number;
  setCommandCount: (count: number) => void;
  onReset: () => void;
  onBusyChange: (busy: boolean) => void;
}) {
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);
  const [hintIndex, setHintIndex] = useState(0);
  const nextLine = useRef(1);

  useEffect(() => {
    setInput("");
    setBusy(false);
    onBusyChange(false);
    setHintIndex(0);
  }, [lesson, onBusyChange]);

  const append = (items: TerminalOutput[]) => {
    setLines((current) => [
      ...current,
      ...items.map((item) => ({ ...item, id: nextLine.current++ })),
    ]);
  };

  const run = async (event: FormEvent) => {
    event.preventDefault();
    const command = input.trim();
    if (!command || busy) return;
    setInput("");
    append([{ kind: "system", text: `$ ${command}` }]);
    const parsed = gitEngine.parse(command);
    if (parsed.error) {
      append([{ kind: "error", text: parsed.error.message }]);
      return;
    }
    const disallowed = parsed.actions.find((action) => !lesson.allowedCommands.includes(actionCommand(action)));
    if (disallowed) {
      append([{ kind: "error", text: `本课暂不开放 ${actionCommand(disallowed)}；输入 help 查看本课边界。` }]);
      return;
    }
    setBusy(true);
    onBusyChange(true);
    let next = state;
    let count = commandCount;
    for (const action of parsed.actions) {
      if (action.kind === "hint") {
        append([{ kind: "system", text: `Hint ${hintIndex + 1}: ${lesson.hints[hintIndex % lesson.hints.length]}` }]);
        setHintIndex((current) => current + 1);
      } else if (action.kind === "goal") {
        append([{ kind: "system", text: `${lesson.goal.summary}\n${lesson.goal.acceptance.map((item) => `• ${item}`).join("\n")}` }]);
      } else if (action.kind === "solution") {
        append([{ kind: "system", text: `Reference solution:\n${lesson.referenceSolution.join("\n")}` }]);
      } else if (action.kind === "reset-runtime") {
        next = cloneState(lesson.startState);
        count = 0;
        append([{ kind: "system", text: "Practice state 已重置。" }]);
      } else {
        const transition = gitEngine.apply(next, action);
        append(transition.output);
        if (transition.error) break;
        next = transition.state;
        if (!["status", "log", "show", "help", "branch"].includes(action.kind)) count += 1;
      }
      setState(next);
      setCommandCount(count);
      await delayForMotion();
    }
    setBusy(false);
    onBusyChange(false);
  };

  return (
    <div className="terminal-shell">
      <div className="terminal-bar">
        <div className="terminal-dots" aria-hidden="true"><i /><i /><i /></div>
        <span>rubickx@simulator · {lesson.id}</span>
        <button onClick={onReset}>Reset practice</button>
      </div>
      <div className="terminal-output" aria-live="polite" aria-label="Terminal output">
        {lines.map((line) => <pre key={line.id} className={`terminal-line terminal-${line.kind}`}>{line.text}</pre>)}
      </div>
      <form className="terminal-input-row" onSubmit={run}>
        <label htmlFor="git-command">$</label>
        <input
          id="git-command"
          value={input}
          onChange={(event) => setInput(event.target.value)}
          placeholder="输入 git status，或 help"
          autoComplete="off"
          spellCheck={false}
          disabled={busy}
          aria-describedby="allowed-commands"
        />
        <button type="submit" disabled={busy}>{busy ? "Running" : "Run"}</button>
      </form>
      <div className="terminal-footer" id="allowed-commands">
        <span>Allowed: {lesson.allowedCommands.join(" · ")}</span>
        <span>{commandCount} state-changing commands</span>
      </div>
    </div>
  );
}

export function GitCourseApp() {
  const initialLessonId = useMemo(() => {
    const fallback = lessons[0].id;
    if (typeof window === "undefined") return fallback;
    const stored = loadProgress(window.localStorage, fallback);
    return lessons.some((lesson) => lesson.id === stored.lastLesson) ? stored.lastLesson : fallback;
  }, []);
  const [lessonId, setLessonId] = useState(initialLessonId);
  const lesson = lessons.find((item) => item.id === lessonId) ?? lessons[0];
  const [progress, setProgress] = useState(() => typeof window === "undefined"
    ? emptyProgress(lessons[0].id)
    : loadProgress(window.localStorage, lessons[0].id));
  const [state, setState] = useState(() => cloneState(lesson.startState));
  const [lines, setLines] = useState<TerminalLine[]>([
    { id: 0, kind: "system", text: "Deterministic Git simulator ready. 输入 goal 查看本课目标。" },
  ]);
  const [commandCount, setCommandCount] = useState(0);
  const [judge, setJudge] = useState<JudgeResult | null>(null);
  const [checkpointChoice, setCheckpointChoice] = useState<number | null>(null);
  const [terminalBusy, setTerminalBusy] = useState(false);

  useEffect(() => {
    saveProgress(window.localStorage, progress);
  }, [progress]);

  const resetPractice = () => {
    setState(cloneState(lesson.startState));
    setLines([{ id: Date.now(), kind: "system", text: "Practice state 已回到 lesson startState。" }]);
    setCommandCount(0);
    setJudge(null);
    setCheckpointChoice(null);
    setTerminalBusy(false);
  };

  const selectLesson = (nextId: string) => {
    const next = lessons.find((item) => item.id === nextId);
    if (!next) return;
    setLessonId(nextId);
    setProgress((current) => ({ ...current, lastLesson: nextId }));
    setState(cloneState(next.startState));
    setLines([{ id: Date.now(), kind: "system", text: `Loaded ${next.title}. 输入 goal 查看目标。` }]);
    setCommandCount(0);
    setJudge(null);
    setCheckpointChoice(null);
    setTerminalBusy(false);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const runJudge = () => {
    const result = judgeRepository(state, targetStateForLesson(lesson), lesson.judgePolicy);
    setJudge(result);
    setProgress((current) => recordAttempt(current, lesson.id, commandCount, result.passed));
  };

  const activePhase = judge?.passed ? 4 : commandCount > 0 ? 3 : 0;
  const checkpointCorrect = checkpointChoice === lesson.checkpoint.correctIndex;

  return (
    <div className="course-app">
      <LessonNavigation selectedId={lesson.id} progress={progress} onSelect={selectLesson} />
      <main className="course-main">
        <header className="lesson-hero">
          <div className="hero-meta"><span>{lesson.eyebrow}</span><span>{lesson.estimatedMinutes} min</span></div>
          <h1>{lesson.title}</h1>
          <p>{lesson.goal.summary}</p>
          <div className="learning-loop" aria-label="Learning loop">
            {["Explain", "Guided demo", "Practice", "Judge", "Checkpoint"].map((phase, index) => (
              <div key={phase} className={index <= activePhase ? "loop-step loop-step-active" : "loop-step"}>
                <span>{String(index + 1).padStart(2, "0")}</span>
                <strong>{phase}</strong>
                <small>{phaseLabel(index, activePhase)}</small>
              </div>
            ))}
          </div>
        </header>

        <section className="lesson-section explain-section" aria-labelledby="explain-title">
          <div className="section-heading">
            <span className="step-chip">01</span>
            <div><span className="section-kicker">Explain</span><h2 id="explain-title">先建立可操作的 mental model</h2></div>
          </div>
          <div className="explain-grid">
            <div className="prose-card">{lesson.explanation.map((paragraph) => <p key={paragraph}>{paragraph}</p>)}</div>
            <div className="goal-card">
              <span className="section-kicker">本课目标 state</span>
              <strong>{lesson.goal.summary}</strong>
              <ul>{lesson.goal.acceptance.map((item) => <li key={item}>{item}</li>)}</ul>
            </div>
          </div>
        </section>

        <GuidedDemo lesson={lesson} />

        <section className="lesson-section practice-section" aria-labelledby="practice-title">
          <div className="section-heading">
            <span className="step-chip">03</span>
            <div><span className="section-kicker">Terminal practice</span><h2 id="practice-title">现在由你改变 repository state</h2></div>
          </div>
          <p className="practice-prompt">{lesson.exercise.prompt}</p>
          <RepositoryView state={state} />
          <PracticeTerminal
            lesson={lesson}
            state={state}
            setState={setState}
            lines={lines}
            setLines={setLines}
            commandCount={commandCount}
            setCommandCount={setCommandCount}
            onReset={resetPractice}
            onBusyChange={setTerminalBusy}
          />
        </section>

        <section className="lesson-section judge-section" aria-labelledby="judge-title">
          <div className="section-heading">
            <span className="step-chip">04</span>
            <div><span className="section-kicker">State-based judge</span><h2 id="judge-title">检查结果，不背固定命令</h2></div>
          </div>
          <div className="judge-card">
            <div>
              <span className="section-kicker">Judge policy</span>
              <p>{lesson.judgePolicy.compare.join(" · ")} · {lesson.judgePolicy.hashAgnostic ? "hash agnostic" : "exact hash"}</p>
            </div>
            <button className="button button-primary" onClick={runJudge} disabled={terminalBusy}>Judge my state</button>
          </div>
          {judge && (
            <div className={judge.passed ? "judge-result judge-pass" : "judge-result judge-retry"} role="status">
              <strong>{judge.passed ? "✓ 目标达成" : "↻ 再观察一次"}</strong>
              <p>{judge.summary}</p>
              {judge.differences.length > 0 && <ul>{judge.differences.map((item) => <li key={item}>{item}</li>)}</ul>}
              {judge.passed && <p>{lesson.exercise.success}</p>}
            </div>
          )}
        </section>

        <section className={judge?.passed ? "lesson-section checkpoint-section" : "lesson-section checkpoint-section checkpoint-locked"} aria-labelledby="checkpoint-title">
          <div className="section-heading">
            <span className="step-chip">05</span>
            <div><span className="section-kicker">Checkpoint</span><h2 id="checkpoint-title">用一句判断固化概念</h2></div>
          </div>
          <fieldset disabled={!judge?.passed}>
            <legend>{lesson.checkpoint.question}</legend>
            {lesson.checkpoint.options.map((option, index) => (
              <button
                key={option}
                className={checkpointChoice === index ? "checkpoint-option checkpoint-option-selected" : "checkpoint-option"}
                onClick={() => setCheckpointChoice(index)}
              >
                <span>{String.fromCharCode(65 + index)}</span>{option}
              </button>
            ))}
          </fieldset>
          {!judge?.passed && <p className="locked-note">完成 Judge 后解锁 checkpoint。</p>}
          {checkpointChoice !== null && (
            <div className={checkpointCorrect ? "checkpoint-feedback checkpoint-correct" : "checkpoint-feedback"}>
              <strong>{checkpointCorrect ? "回答正确" : "再想想 state 的实际位置"}</strong>
              {checkpointCorrect && <p>{lesson.checkpoint.explanation}</p>}
            </div>
          )}
        </section>

        <footer className="lesson-footer">
          <div><span className="section-kicker">Source mapping</span><p>课程表述为原创中文，概念边界映射到 Git 官方资料。</p></div>
          <div className="source-links">
            {lesson.sourceRefs.map((source) => <a key={source.id} href={source.url} target="_blank" rel="noreferrer">{source.title} ↗</a>)}
          </div>
          <small>{gitCourse.title} · Runtime schema v{gitCourse.version} · Gate 1</small>
        </footer>
      </main>
    </div>
  );
}
