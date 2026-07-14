import type {
  Catalog,
  DebugCommand,
  DebugSessionRequest,
  DebugSnapshot,
  DemoOptions,
  DemoResult,
  RunRequest,
  RunResponse,
} from "./types";

interface APIErrorBody {
  error?: {
    code?: string;
    message?: string;
  };
}

interface StopDebugSessionOptions {
  keepalive?: boolean;
}

export class APIError extends Error {
  readonly code: string;

  constructor(message: string, code = "request_failed") {
    super(message);
    this.name = "APIError";
    this.code = code;
  }
}

async function requestJSON<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(input, init);
  } catch (error) {
    throw new APIError(error instanceof Error ? error.message : "Could not reach the lab server", "network_error");
  }

  const body = await response.json().catch(() => undefined) as T | APIErrorBody | undefined;
  if (!response.ok) {
    const apiError = body as APIErrorBody | undefined;
    throw new APIError(
      apiError?.error?.message ?? `Request failed (${response.status})`,
      apiError?.error?.code,
    );
  }
  return body as T;
}

const jsonHeaders = { "Content-Type": "application/json" };

export const api = {
  catalog(signal?: AbortSignal) {
    return requestJSON<Catalog>("/api/catalog", { signal });
  },

  run(payload: RunRequest) {
    return requestJSON<RunResponse>("/api/runs", {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify(payload),
    });
  },

  createDebugSession(payload: DebugSessionRequest) {
    return requestJSON<DebugSnapshot>("/api/debug/sessions", {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify(payload),
    });
  },

  debugCommand(sessionId: string, command: Exclude<DebugCommand, "stop">) {
    return requestJSON<DebugSnapshot>(`/api/debug/sessions/${encodeURIComponent(sessionId)}/commands`, {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify({ command }),
    });
  },

  async stopDebugSession(sessionId: string, options: StopDebugSessionOptions = {}) {
    let response: Response;
    try {
      response = await fetch(`/api/debug/sessions/${encodeURIComponent(sessionId)}`, {
        method: "DELETE",
        ...(options.keepalive ? { keepalive: true } : {}),
      });
    } catch (error) {
      throw new APIError(error instanceof Error ? error.message : "Could not reach the lab server", "network_error");
    }
    if (!response.ok) {
      const body = await response.json().catch(() => undefined) as APIErrorBody | undefined;
      throw new APIError(body?.error?.message ?? `Request failed (${response.status})`, body?.error?.code);
    }
  },

  async demo(endpoint: string, options: DemoOptions): Promise<DemoResult> {
    const query = new URLSearchParams({
      store: options.store,
      failure: options.failure,
      replica: options.replica,
      limit: String(options.limit),
      window_ms: String(options.windowMs),
    });
    const url = `/demo/${encodeURIComponent(endpoint)}?${query}`;
    let response: Response;
    try {
      response = await fetch(url, { headers: { "X-RateLimit-Key": options.clientKey } });
    } catch (error) {
      throw new APIError(error instanceof Error ? error.message : "Could not reach the demo endpoint", "network_error");
    }
    return {
      url,
      key: options.clientKey,
      status: response.status,
      statusText: response.statusText,
      headers: {
        limit: response.headers.get("RateLimit-Limit") ?? "",
        remaining: response.headers.get("RateLimit-Remaining") ?? "",
        reset: response.headers.get("RateLimit-Reset") ?? "",
        retryAfter: response.headers.get("Retry-After") ?? "",
        degraded: response.headers.get("X-RateLimit-Degraded") ?? "",
      },
      body: await response.text(),
    };
  },
};
