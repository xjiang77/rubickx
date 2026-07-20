import type { DemoExchange, DemoOptions } from "./types";
import { useI18n } from "./i18n";

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
  const { t } = useI18n();

  if (conceptual) {
    return (
      <div className="concept-card">
        <span className="eyebrow">{t.tConceptKicker}</span>
        <h3>{t.tConceptTitle}</h3>
        <p>{t.tConceptDesc}</p>
        <div className="concept-balance">
          <span>{t.tLowLatency}</span><i />
          <span>{t.tGlobalAccuracy}</span><i />
          <span>{t.tAvailability}</span>
        </div>
        <small>{t.tConceptNote}</small>
      </div>
    );
  }

  return (
    <form className="system-console" onSubmit={(event) => { event.preventDefault(); onSend(burstPreferred ? 10 : 1); }}>
      <div className="console-intro">
        <span className="eyebrow">{t.tRealHttp}</span>
        <h3>{t.tConsoleTitle}</h3>
        <p>{t.tConsoleDesc}</p>
      </div>

      <div className="console-fields">
        <label>
          <span>{t.tStore}</span>
          <select
            aria-label="Store"
            value={options.store}
            onChange={(event) => onOptions({ ...options, store: event.target.value as DemoOptions["store"] })}
          >
            <option value="memory">{t.tMemory}</option>
            <option value="redis">Redis</option>
          </select>
        </label>
        <label>
          <span>{t.tFailure}</span>
          <select
            aria-label="Failure policy"
            value={options.failure}
            onChange={(event) => onOptions({ ...options, failure: event.target.value as DemoOptions["failure"] })}
          >
            <option value="fail-open">{t.tFailOpen}</option>
            <option value="fail-closed">{t.tFailClosed}</option>
          </select>
        </label>
        {showReplica && (
          <label>
            <span>{t.tReplica}</span>
            <select
              aria-label="Replica"
              value={options.replica}
              onChange={(event) => onOptions({ ...options, replica: event.target.value as DemoOptions["replica"] })}
            >
              <option value="a">{t.tReplicaA}</option>
              <option value="b">{t.tReplicaB}</option>
            </select>
          </label>
        )}
        <label className="client-key-field">
          <span>{t.tClientKey}</span>
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
        <div><dt>{t.tLimit}</dt><dd>{options.limit}</dd></div>
        <div><dt>{t.tWindow}</dt><dd>{options.windowMs} ms</dd></div>
        <div><dt>{t.tTransport}</dt><dd>GET</dd></div>
      </dl>

      <div className="console-actions">
        <button
          className={burstPreferred ? "secondary-action" : "console-primary"}
          type="button"
          disabled={busy || options.clientKey.trim() === ""}
          onClick={() => onSend(1)}
        >
          {t.tSendRequest}
        </button>
        <button
          className={burstPreferred ? "console-primary" : "secondary-action"}
          type="button"
          disabled={busy || options.clientKey.trim() === ""}
          onClick={() => onSend(10)}
        >
          {t.tBurst}
          {burstPreferred && <small>{t.tDefaultBadge}</small>}
        </button>
      </div>
      {busy && <p className="console-progress" role="status">{t.tSendingTraffic}</p>}
    </form>
  );
}

export function SystemRequestLog({ exchanges }: { exchanges: DemoExchange[] }) {
  const { t } = useI18n();

  if (exchanges.length === 0) {
    return (
      <div className="empty-state compact">
        <p>{t.tNoHttp}</p>
        <span>{t.tNoHttpSub}</span>
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
  const { lang, t } = useI18n();

  if (!exchange) {
    return (
      <div className="empty-state compact" aria-label="HTTP exchange">
        <p>{t.tContractHere}</p>
        <span>{t.tContractSub}</span>
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
          <span>{t.tHttpStatus}</span>
          <strong>{exchange.status}</strong>
        </div>
        <p>{lang === "zh"
          ? (exchange.status < 400 ? t.tAllowed : t.tLimited)
          : (exchange.statusText || (exchange.status < 400 ? t.tAllowed : t.tLimited))}</p>
      </div>
      {exchange.headers.degraded === "true" && <div className="degraded-note">{t.tDegraded}</div>}
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
        <h3>{t.tRespBody}</h3>
        <pre>{prettyBody(exchange.body)}</pre>
      </section>
    </div>
  );
}
