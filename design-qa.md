# AI Engineering Skills Map — Design QA

- Source visual truth: `04_Projects/Rubickx/Design/skills-map-selected-sketch.png` in dragon-vault; original `/Users/kevinxjiang/.codex/generated_images/01a061bf-42c1-7851-ab62-4e6c5ee5e7c0/exec-2a5e11c9-0575-45f8-9ac3-e14aebd778fe.png`.
- Local implementation: http://127.0.0.1:8135/index.html
- Implementation screenshot: `/tmp/rubickx-map-desktop-final.png`; full page `/tmp/rubickx-map-desktop-full.png`; mobile `/tmp/rubickx-map-mobile.png`.
- State: Software engineering fundamentals → Designing system architectures; capability/practice directories collapsed.
- Desktop CSS viewport and source pixels: 1386 × 1135, devicePixelRatio 1. Final viewport screenshot: 1386 × 1135; no resampling. Browser returns JPEG bytes despite the temporary `.png` filenames.
- Mobile: 390 × 844 CSS viewport; full-page capture 390 × 2418. It is a responsive adaptation, not a second source mock.

## Findings and comparison history

1. Initial comparison paired the source and `/tmp/rubickx-map-desktop-v1.png` in the same visual input. [P2] Node typography was too heavy; [P2] horizontal connector endpoints did not meet the outer vertical connectors. Reduced node font weights and accounted for grid gaps when calculating connector endpoints.
2. `/tmp/rubickx-map-desktop-v2.png` was 1386 × 739 because the screenshot backend reset its output dimensions. It was rejected as comparison evidence. Reapplied the viewport before capture.
3. Source and `/tmp/rubickx-map-desktop-v3.png` were compared together at 1386 × 1135. Both P2 findings were resolved. Final capture confirms the same layout after source formatting and catalog updates.
4. Mobile full-page capture checked the stacked capability list, complete practice text, readable source links and directory summaries. No horizontal overflow; capability controls are at least 72px tall, the compact extension control about 60px.

## Required fidelity surfaces

- **Typography:** Avenir Next with system Chinese fallback, 62px desktop heading, 16px capability title, 15px secondary label. Source hierarchy and wrapping are retained; browser font rendering differs slightly from the generated image. [P3] The exact generated font is not identifiable; current family/weight is an acceptable system-font match.
- **Spacing/layout:** 48px desktop page padding, four pillar columns, thin connectors and generous graph spacing. The extra Algorithms node adds an intentional extension row after the original five SE capabilities. The panel is taller because the approved plan adds systems-foundations, explicit status, evidence boundaries and cross-links; all content remains available by scrolling.
- **Colors/tokens:** Warm white #fbfbfa, dark #1f2523, light #d9dedb borders, blue/green/purple/orange branches. Orange text/border uses #ad501d for readable contrast. Selection has both border/background and aria-pressed state.
- **Assets:** Official Feather v4.29.2 icons, MIT license included under web/icons. No rasterized UI or handmade substitute icons. Tree connectors are functional CSS layout lines. Source's capability-header illustration is omitted to keep room for the required status and evidence text; this does not alter navigation.
- **Copy/content:** Exact requested title; original 6 AI + 5 SE names/order; three Rubickx extensions explicitly labeled. Nanochat remains scaffold; two placeholders have no source-code button. Git and algorithms disclose limited coverage.

Full-view comparisons are readable at native size, so no focused crop was needed. Full-page and mobile evidence additionally covers panel content below the source viewport.

## Interaction verification

- Browser: four pillar selections; default architecture; Grounding placeholder has zero source-code buttons; Nanochat uses “练习骨架”; local extensions display attribution.
- Browser: Enter and Space selection, all-practices link opens all 11 entries, no main-page console warnings/errors, no mobile horizontal overflow.
- Actual no-JS browser load: QA server sends CSP `script-src 'none'`; all 14 capability descriptions and 11 practices remain visible and linked.
- Actual failed-catalog browser load: QA route returns 404 for capabilities.json; readable fallback and full directories remain available.
- DOM execution of the production map.js with JSDOM: all 14 states, status labels, practice URLs and history reset pass. With reduced-motion=true, animation calls=0; with false, animation calls=15. CSS also disables transitions and animations under prefers-reduced-motion. OS preference was not changed.
- Catalog gate and generated HTML --check pass. Pages basic gate passes.

## Implementation checklist

- [x] Resolve typography and connector findings.
- [x] Validate desktop/mobile/default/placeholder/extension states.
- [x] Validate keyboard, no-JS, fetch failure and reduced-motion branch.
- [x] Preserve complete static navigation from the single capability catalog.
- [ ] Human visual acceptance and stage publication remain pending.

final result: passed
