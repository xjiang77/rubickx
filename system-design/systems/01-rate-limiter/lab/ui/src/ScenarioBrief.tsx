import { useI18n, type Strings } from "./i18n";
import type { ScenarioBriefViewModel } from "./scenarioBriefModel";

export function ScenarioBrief({ model }: { model: ScenarioBriefViewModel }) {
  const { t } = useI18n();
  const hasSequence = model.expected.groups.length > 0;
  const hasCases = model.expected.cases.length > 0;

  return (
    <section className="scenario-brief" aria-label="Scenario brief">
      <header className="scenario-brief-header">
        <span className="eyebrow">{t.tBriefKicker}</span>
        <h2>{t.tBriefTitle}</h2>
      </header>

      <div className="scenario-brief-grid">
        <BriefField title={t.tPolicy} value={model.policy} />
        <BriefField title={t.tTraffic} value={model.traffic} />

        <div className="scenario-brief-field scenario-brief-expected">
          <h3>{t.tExpected}</h3>
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
              <strong>{localizedLessonLabel(model.lesson.label, t)}</strong>
              <span>{model.lesson.text}</span>
            </div>
          )}
          {model.goPath && (
            <div className="scenario-brief-note go-path-note">
              <strong>{t.tGoPath}</strong>
              <span>{model.goPath === "System scenarios run through the Go end-to-end path." ? t.tGoPathNote : model.goPath}</span>
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

function localizedLessonLabel(label: string, t: Strings) {
  if (label === "Concept") return t.tConcept;
  if (label === "Lesson") return t.tLesson;
  return label;
}
