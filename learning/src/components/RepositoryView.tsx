import { describeHead } from "../git/judge";
import { headCommitId, headTree } from "../git/engine";
import type { FileTree, RepositoryState } from "../git/types";

function allPaths(...trees: FileTree[]): string[] {
  return [...new Set(trees.flatMap((tree) => Object.keys(tree)))].sort();
}

function fileState(base: FileTree, current: FileTree, path: string): string {
  if (!(path in current)) return "deleted";
  if (!(path in base)) return "added";
  if (base[path] !== current[path]) return "modified";
  return "unchanged";
}

function FileColumn({ title, kicker, tree, compare }: {
  title: string;
  kicker: string;
  tree: FileTree;
  compare: FileTree;
}) {
  const paths = allPaths(compare, tree);
  return (
    <section className="area-column">
      <div className="area-heading">
        <span>{kicker}</span>
        <h4>{title}</h4>
      </div>
      <ul>
        {paths.map((path) => {
          const state = fileState(compare, tree, path);
          return (
            <li key={path} className={`file-state file-state-${state}`}>
              <span>{path}</span>
              <small>{state}</small>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

function CommitGraph({ state }: { state: RepositoryState }) {
  const commits = Object.values(state.commits).sort((left, right) => {
    const leftNumber = Number(left.id.replace(/\D/g, ""));
    const rightNumber = Number(right.id.replace(/\D/g, ""));
    return leftNumber - rightNumber;
  });
  const position = new Map(commits.map((commit, index) => [commit.id, 52 + index * 112]));
  const width = Math.max(320, 104 + commits.length * 112);
  const current = headCommitId(state);
  return (
    <section className="graph-card" aria-labelledby="graph-title">
      <div className="graph-title-row">
        <div>
          <span className="section-kicker">Repository</span>
          <h4 id="graph-title">Commit graph</h4>
        </div>
        <code>{describeHead(state)}</code>
      </div>
      <div className="graph-scroll">
        <svg viewBox={`0 0 ${width} 126`} role="img" aria-label={`Commit graph，${describeHead(state)}`}>
          {commits.flatMap((commit) => commit.parents.map((parent) => (
            <line
              key={`${commit.id}-${parent}`}
              x1={position.get(parent)}
              x2={position.get(commit.id)}
              y1="62"
              y2="62"
              className="graph-edge"
            />
          )))}
          {commits.map((commit) => {
            const branchLabels = Object.entries(state.branches)
              .filter(([, id]) => id === commit.id)
              .map(([name]) => name);
            const isHead = current === commit.id;
            return (
              <g key={commit.id} transform={`translate(${position.get(commit.id)} 62)`}>
                <circle r="17" className={isHead ? "graph-node graph-node-head" : "graph-node"} />
                <text y="5" textAnchor="middle" className="graph-node-label">{commit.id}</text>
                <text y="38" textAnchor="middle" className="graph-message">{commit.message}</text>
                {branchLabels.length > 0 && (
                  <text y="-29" textAnchor="middle" className="graph-ref">{branchLabels.join(" · ")}</text>
                )}
                {state.head.type === "detached" && state.head.target === commit.id && (
                  <text y="-45" textAnchor="middle" className="graph-head-label">HEAD</text>
                )}
                {state.head.type === "branch" && branchLabels.includes(state.head.target) && (
                  <text y="-45" textAnchor="middle" className="graph-head-label">HEAD</text>
                )}
              </g>
            );
          })}
        </svg>
      </div>
    </section>
  );
}

export function RepositoryView({ state, compact = false }: { state: RepositoryState; compact?: boolean }) {
  const repository = headTree(state);
  return (
    <div className={compact ? "repository-view repository-view-compact" : "repository-view"}>
      <div className="areas-grid">
        <FileColumn title="Working tree" kicker="Edit" tree={state.workingTree} compare={state.index} />
        <div className="area-arrow" aria-hidden="true">add →</div>
        <FileColumn title="Index" kicker="Stage" tree={state.index} compare={repository} />
        <div className="area-arrow" aria-hidden="true">commit →</div>
        <FileColumn title="HEAD snapshot" kicker="Commit" tree={repository} compare={{}} />
      </div>
      {!compact && <CommitGraph state={state} />}
    </div>
  );
}
