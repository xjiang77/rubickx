#!/usr/bin/env python3
"""校验能力目录、证据边界与 README；不把文件存在当成能力完成。"""
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CATALOG = ROOT / 'web/capabilities.json'
ORIGINAL = {
    '01-ai-applications': [
        'LLM foundations', 'Grounding models with data', 'Building agentic systems',
        'Evaluation-driven development', 'Operating in production', 'Machine learning foundations',
    ],
    '02-se-fundamentals': [
        'Building full-stack applications', 'Managing data', 'Designing system architectures',
        'Making systems secure and reliable', 'Scaling and operating in production',
    ],
    '03-coding-agents': [
        'Directing the workflow', 'Enabling agent autonomy', 'Reviewing the work',
        'Customizing the agent and its environment', 'Coding agent foundations',
    ],
    '04-shaping-the-build': [
        'Driving the build loop', 'Making product decisions', 'Communicating and leading',
        'High-agency ownership',
    ],
}
SOURCES = {
    '01-ai-applications': 'https://x.com/AndrewYNg/status/2090840747738374568',
    '02-se-fundamentals': 'https://x.com/AndrewYNg/status/2093388974194872781',
    '03-coding-agents': 'https://x.com/AndrewYNg/status/2095890279865721217',
    '04-shaping-the-build': 'https://x.com/AndrewYNg/status/2098459474608672916',
}


def check(data):
    caps = data['capabilities']
    practices = data['practices']
    assert data['title'] == 'AI Engineering Skills Map'
    assert len(data['pillars']) == 4 and len(caps) == 22 and len(practices) == 11
    ids = {c['id'] for c in caps}
    assert len(ids) == len(caps) and data['defaultCapability'] in ids
    owners = []
    for pillar in data['pillars']:
        group = [c for c in caps if c['pillar'] == pillar['id']]
        assert group, pillar['id']
        assert pillar['source'] == SOURCES[pillar['id']], pillar['id']
        assert [c['title'] for c in group if c['origin'] == 'andrew'] == ORIGINAL[pillar['id']]
        assert {p.name for p in (ROOT / pillar['id']).iterdir() if p.is_dir()} == {c['id'] for c in group}
    for c in caps:
        assert c['path'] == f"{c['pillar']}/{c['id']}"
        readme = ROOT / c['path'] / 'README.md'
        text = readme.read_text()
        assert text.startswith('# ' + c['title'] + '\n'), readme
        assert f"**{data['statuses'][c['status']]}**" in text, readme
        assert c['boundary'] in text and c['next'] in text, readme
        assert c['origin'] in ('andrew', 'rubickx')
        if c['origin'] == 'andrew':
            expected = next(p['source'] for p in data['pillars'] if p['id'] == c['pillar'])
            assert c['source'] == expected and c['source'] in text, readme
        else:
            assert c['source'] is None and 'Rubickx 扩展' in text, readme
        assert all(x in ids for x in c['related'])
        if c['status'] == 'planned':
            assert not c['practices'], c['id']
            assert list((ROOT / c['path']).iterdir()) == [readme], c['id']
        owners.extend(c['practices'])
        for k in c['practices']:
            p = practices[k]
            assert p['path'].startswith(c['path'] + '/'), k
            assert (ROOT / p['path']).exists(), p['path']
            assert p['status'] == c['status'] and p['verify'], k
        for target in re.findall(r'\]\(([^)]+)\)', text):
            if '://' not in target and not target.startswith('#'):
                assert (readme.parent / target.split('#')[0]).exists(), (readme, target)
    assert len(owners) == len(set(owners)) and set(owners) == set(practices)
    assert practices['nanochat']['status'] == 'scaffold'
    print('Capability gate passed: 4 pillars, 22 capabilities (20 original + 2 extensions), 11 unique practices; sources, states, READMEs and links agree.')


if __name__ == '__main__':
    check(json.loads(CATALOG.read_text()))
