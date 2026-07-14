import type { ScenarioBriefViewModel } from "./scenarioBriefModel";

export function ScenarioBrief({ model }: { model: ScenarioBriefViewModel }) {
  const hasSequence = model.expected.groups.length > 0;
  const hasCases = model.expected.cases.length > 0;

  return (
    <section className="scenario-brief" aria-label="Scenario brief">
      <header className="scenario-brief-header">
        <span className="eyebrow">Scenario</span>
        <h2>Scenario brief</h2>
      </header>

      <div className="scenario-brief-grid">
        <BriefField title="Policy" value={model.policy} />
        <BriefField title="Traffic" value={model.traffic} />

        <div className="scenario-brief-field scenario-brief-expected">
          <h3>Expected</h3>
          <p>{model.expected.summary}</p>

          {hasSequence && (
            <ol className="scenario-brief-sequence" aria-label="Expected admission sequence">
              {model.expected.groups.map((group, index) => (
                <li key={`${group.kind}:${group.label}:${index}`} className={`scenario-brief-chip ${group.kind}`}>
                  <strong className="scenario-brief-kind">{decisionLabel(group.kind)}</strong>
                  <span>{group.label}</span>
                </li>
              ))}
            </ol>
          )}

          {hasCases && (
            <ul className="scenario-brief-cases" aria-label="Expected behavior cases">
              {model.expected.cases.map((expectedCase, index) => (
                <li key={`${expectedCase.when}:${expectedCase.kind}:${index}`}>
                  <strong className={`scenario-brief-kind ${expectedCase.kind}`}>
                    {decisionLabel(expectedCase.kind)}
                  </strong>
                  <span className="scenario-brief-case-copy">
                    <span>{expectedCase.when}</span>
                    <span>{expectedCase.result}</span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {(model.lesson || model.goPath) && (
        <footer className="scenario-brief-notes">
          {model.lesson && (
            <div className="scenario-brief-note">
              <strong>{model.lesson.label}</strong>
              <span>{model.lesson.text}</span>
            </div>
          )}
          {model.goPath && (
            <div className="scenario-brief-note go-path-note">
              <strong>Go path</strong>
              <span>{model.goPath}</span>
            </div>
          )}
        </footer>
      )}
    </section>
  );
}

function BriefField({ title, value }: { title: string; value: string }) {
  return (
    <div className="scenario-brief-field">
      <h3>{title}</h3>
      <p>{value}</p>
    </div>
  );
}

function decisionLabel(kind: "allow" | "deny" | "observe") {
  return kind.toUpperCase();
}
