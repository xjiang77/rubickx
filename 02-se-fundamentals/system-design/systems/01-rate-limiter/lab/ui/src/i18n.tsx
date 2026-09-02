import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

export type UiLang = "en" | "zh";

const STORAGE_KEY = "rl-lab-uiLang";

const EN_STRINGS = {
  tTagline: "Run the algorithm. Inspect the state. Follow the source.",
  tLocalLab: "Local lab",
  tScenario: "Scenario",
  tAlgorithm: "Algorithm",
  tLanguage: "Language",
  tMode: "Mode",
  tModeSemantic: "Semantic Trace",
  tModeDebug: "Go Debug",
  tWorking: "Working…",
  tRunScenario: "Run scenario",
  tStartDebug: "Start debug",
  tBriefKicker: "Scenario",
  tBriefTitle: "Scenario brief",
  tPolicy: "Policy",
  tTraffic: "Traffic",
  tExpected: "Expected",
  tLesson: "Lesson",
  tConcept: "Concept",
  tGoPath: "Go path",
  tPanel1Title: "Request timeline",
  tNoTrace: "No trace yet.",
  tNoTraceSub: "Run the scenario to observe each request.",
  tNoHttp: "No HTTP requests yet.",
  tNoHttpSub: "Use the system console to exercise the middleware.",
  tRuntimeState: "Runtime state",
  tSystemConsole: "System console",
  tAlgorithmState: "Algorithm state",
  tHttpExchange: "HTTP exchange",
  tSourceDecision: "Source & decision",
  tCurrentStep: "Current step",
  tRemaining: "Remaining",
  tRetryAfter: "Retry after",
  tResetAt: "Reset at",
  tRawTrace: "Raw trace state",
  tBefore: "Before",
  tAfter: "After",
  tStateHere: "State appears here",
  tStateSub: "Each trace step exposes its before and after snapshots.",
  tSourceFollows: "Source follows execution",
  tSourceSub: "Run a scenario to load the selected implementation.",
  tLine: "Line",
  tConceptKicker: "Conceptual scenario",
  tConceptTitle: "Multi-region quota",
  tConceptDesc: "Compare regional allocation with a strongly consistent global quota.",
  tLowLatency: "Low latency",
  tGlobalAccuracy: "Global accuracy",
  tAvailability: "Availability",
  tConceptNote: "No cluster is simulated. This scenario is a decision exercise.",
  tRealHttp: "Real HTTP middleware",
  tConsoleTitle: "Send traffic through the Go path",
  tConsoleDesc: "Inspect the public rate-limit contract one response at a time.",
  tStore: "Store",
  tFailure: "Failure",
  tMemory: "Memory",
  tFailOpen: "Fail open",
  tFailClosed: "Fail closed",
  tReplica: "Replica",
  tReplicaA: "Replica A",
  tReplicaB: "Replica B",
  tClientKey: "Client key",
  tLimit: "Limit",
  tWindow: "Window",
  tTransport: "Transport",
  tSendRequest: "Send request",
  tBurst: "Burst ×10",
  tDefaultBadge: "default",
  tSendingTraffic: "Sending traffic…",
  tContractHere: "Response contract appears here",
  tContractSub: "Status, quota headers and body come from the real endpoint.",
  tHttpStatus: "HTTP status",
  tAllowed: "Allowed",
  tLimited: "Limited",
  tDegraded: "Store unavailable · degraded response",
  tRespBody: "Response body",
  tDelveReady: "Delve is ready",
  tDelveSub: "Start a Go debug session to inspect its real stack and locals.",
  tNext: "Next",
  tContinue: "Continue",
  tRestart: "Restart",
  tStop: "Stop",
  tLocals: "Locals",
  tNoLocals: "No locals at this stop.",
  tCallStack: "Call stack",
  tPausedAtLine: "Paused at line",
  tDebugSrcUnavailable:
    "Source content is unavailable for this stop. Use the stack and locals to inspect the runtime state.",
  tQuotaKey: "Quota key",
  tTraceTime: "Trace time",
  tCapacity: "Capacity",
  tRequest: "Request",
  tStateUnavailable: "State unavailable",
  tWindowUsed: "Window used",
  tNow: "now",
  tWindowRolled: "Window rolled over",
  tCurrentWindow: "Current fixed window",
  tRollingUsed: "Rolling window used",
  tRollingNow: "rolling now",
  tWeightedUsed: "Weighted window used",
  tEstimating: "estimating…",
  tWeightedPrev: "weighted previous",
  tCurrentLegend: "current",
  tEstimatePending: "Estimate pending after window rotation",
  tPrevious: "previous",
  tAvailTokens: "Available tokens",
  tRefillRate: "Refill rate",
  tRefillCk: "Refill checkpoint",
  tNoTokensConsumed: "No tokens consumed",
  tNoRefill: "No refill needed",
  tQueuedWork: "Queued work",
  tDrainRate: "Drain rate",
  tDrainCk: "Drain checkpoint",
  tQueueUnchanged: "Queue unchanged",
  tNothingToDrain: "Nothing to drain",
  tStepWord: "Step",
  tEventsUnit: "events",
  tRequestsUnit: "requests",
  tFooterLeft: "Deterministic clock · trace replay · real source anchors",
  tLoadingLab: "Loading the lab…",
  tLoadFailed: "Could not load the lab",
  tTryAgain: "Try again",
  tRequestFailed: "Request failed",
  tDelveGoOnly: "Delve DAP is available for Go only.",
  tGoPathNote: "System scenarios run through the Go end-to-end path.",
} as const;

export type StringKey = keyof typeof EN_STRINGS;
export type Strings = Record<StringKey, string>;

export const STRINGS = {
  en: EN_STRINGS,
  zh: {
    tTagline: "运行算法，检查状态，跟随源码。",
    tLocalLab: "本地实验室",
    tScenario: "场景",
    tAlgorithm: "算法",
    tLanguage: "语言",
    tMode: "模式",
    tModeSemantic: "语义追踪",
    tModeDebug: "Go 调试",
    tWorking: "处理中…",
    tRunScenario: "运行场景",
    tStartDebug: "开始调试",
    tBriefKicker: "场景",
    tBriefTitle: "场景简报",
    tPolicy: "策略",
    tTraffic: "流量",
    tExpected: "预期",
    tLesson: "要点",
    tConcept: "概念",
    tGoPath: "Go 链路",
    tPanel1Title: "请求时间轴",
    tNoTrace: "尚无追踪。",
    tNoTraceSub: "运行场景以观察每个请求。",
    tNoHttp: "尚无 HTTP 请求。",
    tNoHttpSub: "使用系统控制台驱动中间件。",
    tRuntimeState: "运行时状态",
    tSystemConsole: "系统控制台",
    tAlgorithmState: "算法状态",
    tHttpExchange: "HTTP 交换",
    tSourceDecision: "源码与决策",
    tCurrentStep: "当前步骤",
    tRemaining: "剩余",
    tRetryAfter: "重试等待",
    tResetAt: "重置时间",
    tRawTrace: "原始追踪状态",
    tBefore: "之前",
    tAfter: "之后",
    tStateHere: "状态将显示在这里",
    tStateSub: "每个追踪步骤都会展示前后两份快照。",
    tSourceFollows: "源码跟随执行",
    tSourceSub: "运行场景以加载所选实现。",
    tLine: "行",
    tConceptKicker: "概念场景",
    tConceptTitle: "多区域配额",
    tConceptDesc: "比较区域分配与强一致的全局配额。",
    tLowLatency: "低延迟",
    tGlobalAccuracy: "全局精确",
    tAvailability: "可用性",
    tConceptNote: "不模拟任何集群。本场景是一次决策练习。",
    tRealHttp: "真实 HTTP 中间件",
    tConsoleTitle: "向 Go 链路发送流量",
    tConsoleDesc: "逐个响应地检查公开的限流契约。",
    tStore: "存储",
    tFailure: "故障策略",
    tMemory: "内存",
    tFailOpen: "失败放行",
    tFailClosed: "失败拒绝",
    tReplica: "副本",
    tReplicaA: "副本 A",
    tReplicaB: "副本 B",
    tClientKey: "客户端 Key",
    tLimit: "限额",
    tWindow: "窗口",
    tTransport: "传输",
    tSendRequest: "发送请求",
    tBurst: "连发 ×10",
    tDefaultBadge: "默认",
    tSendingTraffic: "正在发送流量…",
    tContractHere: "响应契约将显示在这里",
    tContractSub: "状态、配额响应头与响应体来自真实端点。",
    tHttpStatus: "HTTP 状态",
    tAllowed: "允许",
    tLimited: "受限",
    tDegraded: "存储不可用 · 降级响应",
    tRespBody: "响应体",
    tDelveReady: "Delve 已就绪",
    tDelveSub: "启动 Go 调试会话，检查真实的调用栈与局部变量。",
    tNext: "下一步",
    tContinue: "继续",
    tRestart: "重启",
    tStop: "停止",
    tLocals: "局部变量",
    tNoLocals: "此停点无局部变量。",
    tCallStack: "调用栈",
    tPausedAtLine: "暂停于行",
    tDebugSrcUnavailable: "此停点无源码内容。请通过调用栈与局部变量检查运行时状态。",
    tQuotaKey: "配额 Key",
    tTraceTime: "追踪时间",
    tCapacity: "容量",
    tRequest: "请求",
    tStateUnavailable: "状态不可用",
    tWindowUsed: "窗口已用",
    tNow: "当前",
    tWindowRolled: "窗口已滚动",
    tCurrentWindow: "当前固定窗口",
    tRollingUsed: "滚动窗口已用",
    tRollingNow: "滚动当前",
    tWeightedUsed: "加权窗口已用",
    tEstimating: "估算中…",
    tWeightedPrev: "加权上一窗口",
    tCurrentLegend: "当前窗口",
    tEstimatePending: "窗口轮换后估算待定",
    tPrevious: "上一窗口",
    tAvailTokens: "可用令牌",
    tRefillRate: "补充速率",
    tRefillCk: "补充检查点",
    tNoTokensConsumed: "未消耗令牌",
    tNoRefill: "无需补充",
    tQueuedWork: "排队工作",
    tDrainRate: "泄出速率",
    tDrainCk: "泄出检查点",
    tQueueUnchanged: "队列不变",
    tNothingToDrain: "无可泄出",
    tStepWord: "步骤",
    tEventsUnit: "个事件",
    tRequestsUnit: "个请求",
    tFooterLeft: "确定性时钟 · 追踪回放 · 真实源码锚点",
    tLoadingLab: "正在加载实验室…",
    tLoadFailed: "无法加载实验室",
    tTryAgain: "重试",
    tRequestFailed: "请求失败",
    tDelveGoOnly: "Delve DAP 仅支持 Go。",
    tGoPathNote: "System 场景通过 Go 端到端链路运行。",
  },
} satisfies Record<UiLang, Strings>;

interface I18nContextValue {
  lang: UiLang;
  setLang: (lang: UiLang) => void;
  t: Strings;
}

const I18nContext = createContext<I18nContextValue | null>(null);

function persistedLanguage(): UiLang {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored === "en" || stored === "zh" ? stored : "en";
  } catch {
    return "en";
  }
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLang] = useState<UiLang>(persistedLanguage);

  useEffect(() => {
    document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
    try {
      localStorage.setItem(STORAGE_KEY, lang);
    } catch {
      // Language selection remains usable when storage is disabled or full.
    }
  }, [lang]);

  return <I18nContext.Provider value={{ lang, setLang, t: STRINGS[lang] }}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const context = useContext(I18nContext);
  if (!context) {
    throw new Error("useI18n must be used within I18nProvider");
  }
  return context;
}

export function LangToggle() {
  const { lang, setLang } = useI18n();
  return (
    <div className="lang-toggle" role="group" aria-label="Interface language">
      <button type="button" aria-pressed={lang === "en"} onClick={() => setLang("en")}>
        EN
      </button>
      <button type="button" aria-pressed={lang === "zh"} onClick={() => setLang("zh")}>
        中文
      </button>
    </div>
  );
}
