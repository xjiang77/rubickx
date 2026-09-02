import type { CourseDefinition, LessonDefinition, SourceReference } from "../runtime/types";
import { courseSchema } from "../runtime/schema";
import { applyCommandQueue, createRepository } from "./engine";
import type { CommitSnapshot, RepositoryState } from "./types";

const runtimeCommands = ["help", "hint", "goal", "reset", "undo", "solution"];

const officialSources: Record<string, SourceReference> = {
  areas: {
    id: "pro-git-recording-changes",
    title: "Pro Git · Recording Changes",
    url: "https://git-scm.com/book/en/v2/Git-Basics-Recording-Changes-to-the-Repository",
    note: "Working tree、staging area 与 commit snapshot 的官方学习材料。",
  },
  add: {
    id: "git-add-docs",
    title: "git-add Documentation",
    url: "https://git-scm.com/docs/git-add",
    note: "`git add` 如何把 working tree 内容写入 index。",
  },
  commit: {
    id: "git-commit-docs",
    title: "git-commit Documentation",
    url: "https://git-scm.com/docs/git-commit",
    note: "`git commit` 如何从 index 创建新的 commit。",
  },
  revisions: {
    id: "git-revisions-docs",
    title: "gitrevisions Documentation",
    url: "https://git-scm.com/docs/gitrevisions",
    note: "HEAD、branch、relative ref 与 detached HEAD 的精确定义。",
  },
};

function dirtyStart(): RepositoryState {
  const state = createRepository();
  state.workingTree = {
    ...state.workingTree,
    "README.md": "# RubickX\n\nAdd interactive Git notes.\n",
    "app.ts": "export const course = 'interactive-git';\n",
  };
  return state;
}

function graphStart(): RepositoryState {
  const commits: CommitSnapshot[] = [
    {
      id: "C0",
      message: "initial snapshot",
      parents: [],
      tree: { "README.md": "# RubickX\n" },
    },
    {
      id: "C1",
      message: "add course shell",
      parents: ["C0"],
      tree: { "README.md": "# RubickX\n", "course.ts": "export const title = 'Git';\n" },
    },
    {
      id: "C2",
      message: "add lesson metadata",
      parents: ["C1"],
      tree: { "README.md": "# RubickX\n", "course.ts": "export const title = 'Interactive Git';\n" },
    },
  ];
  return createRepository({ commits, branches: { main: "C2" } });
}

const fundamentalsLessons: LessonDefinition<RepositoryState>[] = [
  {
    id: "three-areas",
    sequenceId: "fundamentals",
    order: 1,
    title: "看见 Git 的三区",
    eyebrow: "Lesson 01 · Working tree / Index / Repository",
    estimatedMinutes: 12,
    startState: dirtyStart(),
    goal: {
      summary: "只把 app.ts 放进 index，让 README.md 继续留在 working tree。",
      acceptance: ["app.ts 已 staged", "README.md 仍 unstaged", "没有创建新 commit"],
    },
    explanation: [
      "把 Git 想成三个相邻的工作区：你正在编辑的 working tree、准备进入下一次 commit 的 index，以及已经固化的 repository snapshots。",
      "`git add` 不是把文件“加入 Git”这么模糊；它把指定 path 此刻的内容复制进 index。随后继续编辑同一个文件，会再次产生 unstaged change。",
    ],
    demoSteps: [
      {
        id: "inspect-dirty-state",
        title: "先读 state",
        command: "git status",
        explanation: "status 比较 HEAD ↔ index 与 index ↔ working tree，两段差异分别对应 staged 和 unstaged。",
        focus: "三区面板中的两条变化路径",
      },
      {
        id: "stage-one-path",
        title: "只移动一个 snapshot",
        command: "git add app.ts",
        explanation: "app.ts 的 working tree 内容被复制到 index；README.md 没有移动。",
        focus: "app.ts 从 unstaged 变为 staged",
      },
      {
        id: "inspect-split-state",
        title: "验证选择性 staging",
        command: "git status",
        explanation: "同一个 repository state 可以同时包含 staged 与 unstaged changes。",
        focus: "status 的两个 section",
      },
    ],
    exercise: {
      prompt: "当前 app.ts 和 README.md 都改过。只 stage app.ts，然后让 Judge 检查三区 state。",
      success: "你已经精确控制了下一次 commit 会看到的 snapshot。",
      starterCommands: ["git status"],
    },
    checkpoint: {
      question: "执行 `git add app.ts` 后，又继续修改 app.ts。此时最准确的描述是什么？",
      options: [
        "index 与 working tree 都自动保持最新",
        "index 保留 add 时的版本，新的修改仍是 unstaged",
        "repository 中已经生成一个 commit",
      ],
      correctIndex: 1,
      explanation: "Index 是显式更新的 snapshot；新的 working tree 修改不会自动进入 index。",
    },
    referenceSolution: ["git add app.ts"],
    hints: ["先用 `git status` 区分两类变化。", "`git add` 可以只接收一个 path。"],
    judgePolicy: { compare: ["topology", "refs", "index", "workingTree"], hashAgnostic: true },
    sourceRefs: [officialSources.areas, officialSources.add],
    allowedCommands: ["status", "add", ...runtimeCommands],
  },
  {
    id: "stage-and-commit",
    sequenceId: "fundamentals",
    order: 2,
    title: "用 Index 设计 Commit",
    eyebrow: "Lesson 02 · Stage & Commit",
    estimatedMinutes: 15,
    startState: dirtyStart(),
    goal: {
      summary: "把 app.ts 与 README.md 拆成两个 commit，最后保持 clean。",
      acceptance: ["新增两个线性 commit", "每个 commit 只引入一个主题", "working tree 与 index clean"],
    },
    explanation: [
      "Commit 不是把 working tree 整体存起来；它读取 index 并创建一个 immutable snapshot。选择性 `add` 让一次 commit 只表达一个清晰意图。",
      "小而聚焦的 commit 更容易 review、revert 与 cherry-pick。这里先练习组织 snapshot，message 的措辞不会影响 Judge。",
    ],
    demoSteps: [
      {
        id: "stage-code",
        title: "先准备 code snapshot",
        command: "git add app.ts",
        explanation: "Index 现在只包含 app.ts 的新版本。",
        focus: "Index 与 working tree 的 file tree 不同",
      },
      {
        id: "commit-code",
        title: "从 Index 创建 commit",
        command: "git commit -m \"update course\"",
        explanation: "新 commit 的 tree 等于执行 commit 前的 index。README.md 仍留在 working tree。",
        focus: "Graph 新增一个 node，main 向前移动",
      },
      {
        id: "inspect-after-commit",
        title: "读剩余 change",
        command: "git status",
        explanation: "第一次 commit 后只剩 README.md unstaged，说明 snapshot 边界清晰。",
        focus: "Working tree 中的 README.md",
      },
    ],
    exercise: {
      prompt: "先提交 app.ts，再单独提交 README.md。完成后 working tree 必须 clean。",
      success: "两个 focused commits 组成了可读的线性历史。",
      starterCommands: ["git status", "git log"],
    },
    checkpoint: {
      question: "`git commit` 默认从哪里读取要保存的 file snapshot？",
      options: ["Working tree", "Index", "最新的 remote branch"],
      correctIndex: 1,
      explanation: "Commit 固化的是 index，而不是 working tree 中所有最新内容。",
    },
    referenceSolution: [
      "git add app.ts",
      "git commit -m \"update course\"",
      "git add README.md",
      "git commit -m \"document course\"",
    ],
    hints: ["每次 commit 前只 add 一个 path。", "第二次 commit 前用 status 确认只剩 README.md。"],
    judgePolicy: { compare: ["topology", "refs", "trees", "index", "workingTree"], hashAgnostic: true },
    sourceRefs: [officialSources.areas, officialSources.add, officialSources.commit],
    allowedCommands: ["status", "add", "commit", "log", "show", ...runtimeCommands],
  },
  {
    id: "commit-graph-head",
    sequenceId: "fundamentals",
    order: 3,
    title: "沿 Commit Graph 移动 HEAD",
    eyebrow: "Lesson 03 · Commit Graph / HEAD",
    estimatedMinutes: 14,
    startState: graphStart(),
    goal: {
      summary: "让 main 留在 C2，同时把 HEAD detached 到它的第一个 parent。",
      acceptance: ["main 仍指向 C2", "HEAD detached at C1", "working tree 匹配 C1 snapshot"],
    },
    explanation: [
      "Commit graph 由 commit nodes 与 parent edges 构成。Branch 只是一个可移动的名字；HEAD 通常指向当前 branch，也可以直接指向某个 commit。",
      "`HEAD^` 表示 HEAD commit 的第一个 parent。使用 `switch --detach` 只移动 HEAD，不会把 main 一起带走，适合临时查看旧 snapshot。",
    ],
    demoSteps: [
      {
        id: "read-graph",
        title: "从 HEAD 沿 parent 回看",
        command: "git log",
        explanation: "Log 从 HEAD 开始沿 first-parent 展开，因此顺序是 C2、C1、C0。",
        focus: "SVG graph 的 node 与 parent edge",
      },
      {
        id: "inspect-parent",
        title: "解析 relative ref",
        command: "git show HEAD^",
        explanation: "HEAD^ 被解析为 C1，但 show 只读取，不改变 state。",
        focus: "C1 的 immutable file snapshot",
      },
      {
        id: "detach-head",
        title: "只移动 HEAD",
        command: "git switch --detach HEAD^",
        explanation: "HEAD 直接指向 C1；main 仍留在 C2。",
        focus: "HEAD 与 main 标签分离",
      },
    ],
    exercise: {
      prompt: "使用 relative ref 回到 main 的上一个 commit，但不要移动 main。",
      success: "你已经把 branch ref 与 HEAD 的角色拆开理解。",
      starterCommands: ["git log", "git show HEAD^"],
    },
    checkpoint: {
      question: "Detached HEAD 最准确的含义是什么？",
      options: ["HEAD 直接指向 commit，而不是指向 branch", "main branch 被删除", "Working tree 不再受 Git 管理"],
      correctIndex: 0,
      explanation: "Detached HEAD 是正常的查看/实验状态；branch refs 可以留在原位。",
    },
    referenceSolution: ["git switch --detach HEAD^"],
    hints: ["`HEAD^` 表示第一个 parent。", "要保持 main 不动，需要 `--detach`。"],
    judgePolicy: { compare: ["topology", "refs", "index", "workingTree"], hashAgnostic: true },
    sourceRefs: [officialSources.revisions],
    allowedCommands: ["status", "log", "show", "branch", "switch", ...runtimeCommands],
  },
];

export const gitCourse: CourseDefinition<RepositoryState> = {
  id: "interactive-git",
  version: 1,
  title: "Interactive Git Course",
  description: "在可观察、可撤销的 repository state 中建立 Git 工程心智模型。",
  locale: "zh-CN",
  sequences: [
    {
      id: "fundamentals",
      order: 1,
      title: "Sequence 1 · Fundamentals",
      description: "从三区模型进入 commit graph，理解每条 command 究竟移动了什么。",
      lessons: fundamentalsLessons,
    },
  ],
};

courseSchema.parse(gitCourse);

export const lessons = gitCourse.sequences.flatMap((sequence) => sequence.lessons);

export function targetStateForLesson(lesson: LessonDefinition<RepositoryState>): RepositoryState {
  let state = structuredClone(lesson.startState);
  for (const command of lesson.referenceSolution) {
    const transition = applyCommandQueue(state, command);
    if (transition.error) throw new Error(`${lesson.id}: ${transition.error.message}`);
    state = transition.state;
  }
  return state;
}
