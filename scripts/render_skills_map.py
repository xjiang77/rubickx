#!/usr/bin/env python3
"""从发布的能力目录生成无 JavaScript 也可读的首页内容。"""
import argparse
import html
import json
from pathlib import Path
from urllib.parse import quote

ROOT = Path(__file__).resolve().parents[1]
BASE = 'https://github.com/xjiang77/rubickx/tree/main/'


def render():
    data = json.loads((ROOT / 'web/capabilities.json').read_text())
    esc = html.escape
    groups = []
    for pillar in data['pillars']:
        lines = [f'<section class="directory-group"><h2>{esc(pillar["title"])}</h2>']
        for c in data['capabilities']:
            if c['pillar'] != pillar['id']:
                continue
            source = 'Andrew Ng 原框架' if c['origin'] == 'andrew' else 'Rubickx 扩展'
            lines.append(f'<article id="directory-{c["id"]}"><h3><a href="#directory-{c["id"]}" data-capability="{c["id"]}">{esc(c["title"])}</a> / {esc(c["zh"])}</h3><p class="meta">{source} · {data["statuses"][c["status"]]}</p><p>{esc(c["description"])}</p><p>{esc(c["boundary"])}</p><a href="{BASE}{quote(c["path"] + "/README.md")}">能力说明 README</a></article>')
        groups.append('\n'.join(lines) + '</section>')
    practices = []
    for p in data['practices'].values():
        practices.append(f'<li><a href="{BASE}{quote(p["path"])}"><code>{p["id"]}</code></a><span>{esc(p["description"])} · {data["statuses"][p["status"]]}</span></li>')
    template = (ROOT / 'web/page.template.html').read_text()
    return template.replace('<!-- CAPABILITIES -->', '\n'.join(groups)).replace('<!-- PRACTICES -->', '<ul class="all-practice-list">' + '\n'.join(practices) + '</ul>')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    output = ROOT / 'web/index.html'
    rendered = render()
    if args.check:
        if output.read_text() != rendered:
            raise SystemExit('Homepage stale: run python3 scripts/render_skills_map.py')
        print('Static fallback matches capability catalog.')
    else:
        output.write_text(rendered)
        print('Rendered web/index.html from capability catalog.')
