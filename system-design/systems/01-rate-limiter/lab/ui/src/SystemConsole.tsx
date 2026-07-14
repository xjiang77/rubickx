import type { DemoExchange, DemoOptions } from "./types";

interface SystemConsoleProps {
  options: DemoOptions;
  busy: boolean;
  conceptual: boolean;
  burstPreferred: boolean;
  showReplica: boolean;
  onOptions: (options: DemoOptions) => void;
  onSend: (count: number) => void;
}

export function SystemConsole({
  options,
  busy,
  conceptual,
  burstPreferred,
  showReplica,
  onOptions,
  onSend,
}: SystemConsoleProps) {
  if (conceptual) {
    return (
      <div className="concept-card">
        <span className="eyebrow">Conceptual scenario</span>
        <h3>Multi-region quota</h3>
        <p>Compare regional allocation with a strongly consistent global quota.</p>
        <div className="concept-balance">
          <span>Low latency</span><i />
          <span>Global accuracy</span><i />
          <span>Availability</span>
        </div>
        <small>No cluster is simulated. This scenario is a decision exercise.</small>
      </div>
    );
  }

  return (
    <form className="system-console" onSubmit={(event) => { event.preventDefault(); onSend(burstPreferred ? 10 : 1); }}>
      <div className="console-intro">
        <span className="eyebrow">Real HTTP middleware</span>
        <h3>Send traffic through the Go path</h3>
        <p>Inspect the public rate-limit contract one response at a time.</p>
      </div>

      <div className="console-fields">
        <label>
          <span>Store</span>
          <select
            aria-label="Store"
            value={options.store}
            onChange={(event) => onOptions({ ...options, store: event.target.value as DemoOptions["store"] })}
          >
            <option value="memory">Memory</option>
            <option value="redis">Redis</option>
          </select>
        </label>
        <label>
          <span>Failure</span>
          <select
            aria-label="Failure policy"
            value={options.failure}
            onChange={(event) => onOptions({ ...options, failure: event.target.value as DemoOptions["failure"] })}
          >
            <option value="fail-open">Fail open</option>
            <option value="fail-closed">Fail closed</option>
          </select>
        </label>
        {showReplica && (
          <label>
            <span>Replica</span>
            <select
              aria-label="Replica"
              value={options.replica}
              onChange={(event) => onOptions({ ...options, replica: event.target.value as DemoOptions["replica"] })}
            >
              <option value="a">Replica A</option>
              <option value="b">Replica B</option>
            </select>
          </label>
        )}
        <label className="client-key-field">
          <span>Client key</span>
          <input
            aria-label="Client key"
            value={options.clientKey}
            maxLength={128}
            spellCheck={false}
            onChange={(event) => onOptions({ ...options, clientKey: event.target.value })}
          />
        </label>
      </div>

      <dl className="policy-summary">
        <div><dt>Limit</dt><dd>{options.limit}</dd></div>
        <div><dt>Window</dt><dd>{options.windowMs} ms</dd></div>
        <div><dt>Transport</dt><dd>GET</dd></div>
      </dl>

      <div className="console-actions">
        <button
          className={burstPreferred ? "secondary-action" : "console-primary"}
          type="button"
          disabled={busy || options.clientKey.trim() === ""}
          onClick={() => onSend(1)}
        >
          Send request
        </button>
        <button
          className={burstPreferred ? "console-primary" : "secondary-action"}
          type="button"
          disabled={busy || options.clientKey.trim() === ""}
          onClick={() => onSend(10)}
        >
          Burst ×10
          {burstPreferred && <small>default</small>}
        </button>
      </div>
      {busy && <p className="console-progress" role="status">Sending traffic…</p>}
    </form>
  );
}

export function SystemRequestLog({ exchanges }: { exchanges: DemoExchange[] }) {
  if (exchanges.length === 0) {
    return (
      <div className="empty-state compact">
        <p>No HTTP requests yet.</p>
        <span>Use the system console to exercise the middleware.</span>
      </div>
    );
  }

  return (
    <ol className="http-log" aria-label="HTTP request log">
      {exchanges.map((exchange, index) => (
        <li key={exchange.id} className={exchange.status >= 400 ? "rejected" : "accepted"}>
          <span className="request-number">{String(index + 1).padStart(2, "0")}</span>
          <div>
            <strong>GET <span>{exchange.status}</span></strong>
            <small>{exchange.key}</small>
          </div>
          <code>{exchange.headers.remaining || "—"}</code>
        </li>
      ))}
    </ol>
  );
}

function prettyBody(body: string) {
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body || "(empty body)";
  }
}

export function SystemExchange({ exchange }: { exchange?: DemoExchange }) {
  if (!exchange) {
    return (
      <div className="empty-state compact" aria-label="HTTP exchange">
        <p>Response contract appears here</p>
        <span>Status, quota headers and body come from the real endpoint.</span>
      </div>
    );
  }

  const headerRows = [
    ["RateLimit-Limit", exchange.headers.limit],
    ["RateLimit-Remaining", exchange.headers.remaining],
    ["RateLimit-Reset", exchange.headers.reset],
    ["Retry-After", exchange.headers.retryAfter],
  ];

  return (
    <div className="http-exchange" aria-label="HTTP exchange">
      <div className={`http-status ${exchange.status >= 400 ? "rejected" : "accepted"}`}>
        <div>
          <span>HTTP status</span>
          <strong>{exchange.status}</strong>
        </div>
        <p>{exchange.statusText || (exchange.status < 400 ? "Allowed" : "Limited")}</p>
      </div>
      {exchange.headers.degraded === "true" && <div className="degraded-note">Store unavailable · degraded response</div>}
      <div className="request-contract">
        <span>{exchange.url}</span>
        <code>X-RateLimit-Key: {exchange.key}</code>
      </div>
      <dl className="header-list">
        {headerRows.map(([name, value]) => (
          <div key={name}>
            <dt>{name}</dt>
            <dd>{value || "—"}</dd>
          </div>
        ))}
      </dl>
      <section className="response-body">
        <h3>Response body</h3>
        <pre>{prettyBody(exchange.body)}</pre>
      </section>
    </div>
  );
}
