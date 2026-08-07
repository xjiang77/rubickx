/**
 * Minimal local observability for pi agent runs.
 *
 * Writes one JSONL file per process under `.pi/obs/`:
 *   - agent / turn lifecycle with duration_ms
 *   - provider HTTP response status + request timing
 *   - tool start/end with duration_ms and truncated args
 *   - assistant usage (tokens + cost) on turn_end
 *
 * Commands:
 *   /obs       show current log path + short counters
 *   /obs open  print absolute log path (for `tail -f`)
 */
import { appendFileSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";

type JsonRecord = Record<string, unknown>;

const MAX_ARG_CHARS = 500;
const MAX_RESULT_CHARS = 300;

function nowIso(): string {
	return new Date().toISOString();
}

function truncate(value: unknown, maxChars: number): unknown {
	if (typeof value === "string") {
		return value.length <= maxChars ? value : `${value.slice(0, maxChars)}…`;
	}
	try {
		const text = JSON.stringify(value);
		if (text === undefined) return String(value);
		return text.length <= maxChars ? value : `${text.slice(0, maxChars)}…`;
	} catch {
		return "[unserializable]";
	}
}

function modelInfo(ctx: ExtensionContext): JsonRecord | undefined {
	const model = ctx.model;
	if (!model) return undefined;
	return {
		provider: model.provider,
		id: model.id,
		api: model.api,
	};
}

function usageFromMessage(message: unknown): JsonRecord | undefined {
	if (!message || typeof message !== "object") return undefined;
	const msg = message as {
		role?: string;
		usage?: JsonRecord;
		stopReason?: string;
		provider?: string;
		model?: string;
	};
	if (msg.role !== "assistant" || !msg.usage) return undefined;
	const usage = msg.usage;
	const cost = usage.cost && typeof usage.cost === "object" ? (usage.cost as JsonRecord) : undefined;
	return {
		input: usage.input,
		output: usage.output,
		cacheRead: usage.cacheRead,
		cacheWrite: usage.cacheWrite,
		reasoning: usage.reasoning,
		totalTokens: usage.totalTokens,
		costTotal: cost?.total,
		stopReason: msg.stopReason,
		provider: msg.provider,
		model: msg.model,
	};
}

export default function (pi: ExtensionAPI) {
	const startedAt = Date.now();
	const runId = `obs-${startedAt}-${process.pid}`;
	let logPath: string | undefined;
	let seq = 0;

	const counters = {
		agentStarts: 0,
		turns: 0,
		tools: 0,
		toolErrors: 0,
		providerResponses: 0,
	};

	let agentStartedAt: number | undefined;
	const turnStartedAt = new Map<number, number>();
	const toolStartedAt = new Map<string, number>();
	let providerRequestStartedAt: number | undefined;

	function ensureLogPath(ctx: ExtensionContext): string {
		if (logPath) return logPath;
		const dir = join(ctx.cwd, ".pi", "obs");
		mkdirSync(dir, { recursive: true });
		const stamp = new Date(startedAt).toISOString().replaceAll(":", "").replaceAll(".", "-");
		logPath = join(dir, `${stamp}-${process.pid}.jsonl`);
		return logPath;
	}

	function write(ctx: ExtensionContext, event: string, data: JsonRecord = {}): void {
		const path = ensureLogPath(ctx);
		const record = {
			ts: nowIso(),
			seq: ++seq,
			runId,
			event,
			sessionId: ctx.sessionManager.getSessionId(),
			cwd: ctx.cwd,
			model: modelInfo(ctx),
			...data,
		};
		appendFileSync(path, `${JSON.stringify(record)}\n`, "utf8");
	}

	pi.on("session_start", async (_event, ctx) => {
		write(ctx, "session_start", { logPath: ensureLogPath(ctx) });
		ctx.ui.setStatus("obs", `obs → ${ensureLogPath(ctx)}`);
	});

	pi.on("session_shutdown", async (_event, ctx) => {
		write(ctx, "session_shutdown", {
			uptimeMs: Date.now() - startedAt,
			counters,
			logPath: ensureLogPath(ctx),
		});
		ctx.ui.setStatus("obs", undefined);
	});

	pi.on("agent_start", async (_event, ctx) => {
		agentStartedAt = Date.now();
		counters.agentStarts += 1;
		write(ctx, "agent_start");
	});

	pi.on("agent_end", async (event, ctx) => {
		write(ctx, "agent_end", {
			durationMs: agentStartedAt === undefined ? undefined : Date.now() - agentStartedAt,
			messageCount: event.messages.length,
		});
		agentStartedAt = undefined;
	});

	pi.on("agent_settled", async (_event, ctx) => {
		write(ctx, "agent_settled", { counters });
	});

	pi.on("turn_start", async (event, ctx) => {
		turnStartedAt.set(event.turnIndex, Date.now());
		counters.turns += 1;
		write(ctx, "turn_start", { turnIndex: event.turnIndex });
	});

	pi.on("turn_end", async (event, ctx) => {
		const started = turnStartedAt.get(event.turnIndex);
		turnStartedAt.delete(event.turnIndex);
		write(ctx, "turn_end", {
			turnIndex: event.turnIndex,
			durationMs: started === undefined ? undefined : Date.now() - started,
			toolResultCount: event.toolResults.length,
			usage: usageFromMessage(event.message),
		});
	});

	pi.on("before_provider_request", (event, ctx) => {
		providerRequestStartedAt = Date.now();
		let payloadBytes: number | undefined;
		try {
			payloadBytes = Buffer.byteLength(JSON.stringify(event.payload), "utf8");
		} catch {
			payloadBytes = undefined;
		}
		write(ctx, "provider_request", { payloadBytes });
	});

	pi.on("after_provider_response", (event, ctx) => {
		counters.providerResponses += 1;
		write(ctx, "provider_response", {
			status: event.status,
			durationMs: providerRequestStartedAt === undefined ? undefined : Date.now() - providerRequestStartedAt,
			retryAfter: event.headers["retry-after"] ?? event.headers["Retry-After"],
		});
		providerRequestStartedAt = undefined;
	});

	pi.on("tool_execution_start", async (event, ctx) => {
		toolStartedAt.set(event.toolCallId, Date.now());
		counters.tools += 1;
		write(ctx, "tool_start", {
			toolCallId: event.toolCallId,
			toolName: event.toolName,
			args: truncate(event.args, MAX_ARG_CHARS),
		});
	});

	pi.on("tool_execution_end", async (event, ctx) => {
		const started = toolStartedAt.get(event.toolCallId);
		toolStartedAt.delete(event.toolCallId);
		if (event.isError) counters.toolErrors += 1;
		write(ctx, "tool_end", {
			toolCallId: event.toolCallId,
			toolName: event.toolName,
			isError: event.isError,
			durationMs: started === undefined ? undefined : Date.now() - started,
			result: truncate(event.result, MAX_RESULT_CHARS),
		});
	});

	pi.on("model_select", async (event, ctx) => {
		write(ctx, "model_select", {
			source: event.source,
			from: event.previousModel
				? { provider: event.previousModel.provider, id: event.previousModel.id }
				: undefined,
			to: { provider: event.model.provider, id: event.model.id },
		});
	});

	pi.registerCommand("obs", {
		description: "Show local observability JSONL path and counters",
		handler: async (args, ctx) => {
			const path = ensureLogPath(ctx);
			const trimmed = args.trim();
			if (trimmed === "open" || trimmed === "path") {
				ctx.ui.notify(path, "info");
				return;
			}
			ctx.ui.notify(
				[
					`log: ${path}`,
					`agents=${counters.agentStarts} turns=${counters.turns} tools=${counters.tools} toolErrors=${counters.toolErrors} providerResponses=${counters.providerResponses}`,
					`tail -f ${path}`,
				].join("\n"),
				"info",
			);
		},
	});
}
