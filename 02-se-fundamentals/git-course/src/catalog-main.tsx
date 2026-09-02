import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

function Catalog() {
  return (
    <main className="catalog-page">
      <a className="brand catalog-brand" href="../">
        <span className="brand-mark">RX</span>
        <span><strong>RubickX</strong><small>Learning Runtime</small></span>
      </a>
      <section className="catalog-hero">
        <span className="section-kicker">Learn by changing state</span>
        <h1>别只读概念。<br />让系统在你手里运行。</h1>
        <p>每个 learning project 都遵循 Explain → Guided Demo → Practice → State-based Judge → Checkpoint。</p>
      </section>
      <section className="catalog-grid">
        <a className="course-tile" href="./git/">
          <div className="course-tile-top"><span>Now learning</span><span>Gate 1 / 4</span></div>
          <div className="course-icon">git</div>
          <h2>Interactive Git Course</h2>
          <p>在 deterministic simulator 里观察 working tree、index、commit graph 和 HEAD。</p>
          <div className="course-tags"><span>中文优先</span><span>3 lessons live</span><span>Pure browser</span></div>
          <strong className="course-cta">进入课程 <span>→</span></strong>
        </a>
        <div className="course-tile course-tile-planned">
          <div className="course-tile-top"><span>Runtime ready</span><span>Next project</span></div>
          <div className="course-icon">＋</div>
          <h2>Reusable by design</h2>
          <p>同一套 Course schema、transition trace、judge 与 progress contract 会承载后续 learning projects。</p>
          <div className="course-tags"><span>TypeScript</span><span>Zod</span><span>State judge</span></div>
          <strong className="course-cta">Coming after Git</strong>
        </div>
      </section>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<StrictMode><Catalog /></StrictMode>);
