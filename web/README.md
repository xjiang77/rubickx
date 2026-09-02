# AI Engineering Skills Map static site

`capabilities.json` is the published capability catalog. It drives the interactive graph and the generated no-JavaScript directory. Update the relevant capability README whenever classification, status or evidence boundaries change.

From the repo root:

```sh
python3 scripts/check_capabilities.py
python3 scripts/render_skills_map.py
python3 scripts/render_skills_map.py --check
bash .github/scripts/check-pages.sh
python3 -m http.server 8135 --bind 127.0.0.1 -d web
```

Edit `page.template.html`, `styles.css` and `map.js`; regenerate `index.html` after template/catalog changes. GitHub Pages uploads the existing `web/` directory. No frontend framework, remote runtime scripts or build dependency is required.

The README-only placeholders remain browsable. Status expresses repository evidence, not proficiency. When JavaScript or catalog loading is unavailable, the generated HTML keeps all capability and practice links. Source links target `main` and become available after the capability migration is merged.

Icons: Feather v4.29.2, MIT; see `icons/LICENSE`. Design decisions and source cards are maintained in the Rubickx project in dragon-vault; local visual QA is recorded in the repo-root `design-qa.md`.
