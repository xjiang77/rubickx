#!/usr/bin/env python3
"""校验 topic 目录：每个能力都有 topic，状态与证据路径一致，ELI5 页回链到 topic。

topics.json 是 Skills Map 落地计划的登记表：一个 topic 由知识正文（私有 vault）、
ELI5 页（web/learn/）与实践（仓库内路径）三部分组成。这里只检查仓库能证明的事：
路径存在、状态字段自洽、页面与登记互相指向；不把文件存在当成理解或掌握。
"""
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TOPICS = ROOT / 'web/topics.json'
CAPS = ROOT / 'web/capabilities.json'
ID = re.compile(r'^[1-4]\.[1-6]\.[0-9]$')
SLUG = re.compile(r'^[a-z0-9]+(-[a-z0-9]+)*$')


def check(doc, caps):
    states = doc['states']
    cap_ids = {c['id']: c for c in caps['capabilities']}
    topics = doc['topics']
    ids = [t['id'] for t in topics]
    slugs = [t['slug'] for t in topics]
    assert len(ids) == len(set(ids)), 'duplicate topic id'
    assert len(slugs) == len(set(slugs)), 'duplicate topic slug'
    covered = set()
    for t in topics:
        where = t['id']
        assert ID.match(t['id']) and SLUG.match(t['slug']), where
        assert t['capability'] in cap_ids, where
        cap = cap_ids[t['capability']]
        pillar_no = int(cap['pillar'][:2])
        cap_no = int(cap['id'][:2])
        assert t['id'].startswith(f'{pillar_no}.{cap_no}.'), (where, 'id 与能力编号不一致')
        assert t['title'].strip(), where
        covered.add(t['capability'])

        k = t['knowledge']
        assert k['state'] in states['knowledge'], where
        assert (k['state'] == 'missing') == (not k['notes']), (where, '缺正文与 notes 为空必须一致')
        assert all(n.startswith('02_Knowledge/') and n.endswith('.md') for n in k['notes']), where

        p = t['practice']
        assert p['state'] in states['practice'] and p['kind'] in ('code', 'tool', 'case-study'), where
        if p['state'] == 'missing':
            assert not p['paths'] and p['plan'], (where, '待建实践要写计划、不能挂路径')
        else:
            assert p['paths'] and p['verify'], (where, '已有实践要有路径与验证命令')
            for path in p['paths']:
                assert (ROOT / path).exists(), (where, path)
        if p['state'] == 'partial':
            assert p['plan'], (where, '部分实践要写缺口')

        e = t['eli5']
        if e is not None:
            page = ROOT / e['path']
            assert e['path'] == f"web/learn/{t['capability']}/{t['slug']}.html", where
            assert page.exists(), (where, e['path'])
            html = page.read_text()
            assert f'<meta name="rubickx-topic" content="{t["id"]}">' in html, (where, 'ELI5 页缺 topic 元数据')
            src = e['derived_from']
            assert src['note'] in k['notes'] and re.match(r'^\d{4}-\d{2}-\d{2}$', src['updated']), where
            assert f'<meta name="rubickx-derived-from" content="{src["note"]}@{src["updated"]}">' in html, where
    ids_set = set(ids)
    for t in topics:
        rel = t.get('relations')
        if rel is None:
            continue
        assert set(rel) == {'requires', 'extends', 'applies_to'}, (t['id'], 'relations 须含 requires / extends / applies_to')
        for kind, targets in rel.items():
            for x in targets:
                assert x in ids_set and x != t['id'], (t['id'], kind, x)
    missing = set(cap_ids) - covered
    assert not missing, ('能力没有 topic', sorted(missing))
    # 页面目录里的每个 ELI5 页都必须登记
    registered = {t['eli5']['path'] for t in topics if t['eli5']}
    on_disk = {str(p.relative_to(ROOT)) for p in (ROOT / 'web/learn').rglob('*.html')
               if not p.name.startswith('_')} if (ROOT / 'web/learn').exists() else set()
    assert on_disk == registered, ('ELI5 页与登记不一致', sorted(on_disk ^ registered))
    count = lambda key, field: sum(1 for t in topics if (t[key] or {}).get('state', 'present' if t[key] else 'missing') == field)
    eli5 = sum(1 for t in topics if t['eli5'])
    print(f"Topic gate passed: {len(topics)} topics across {len(covered)} capabilities; "
          f"knowledge verified {count('knowledge', 'verified')}, ELI5 {eli5}, "
          f"practice present {count('practice', 'present')} / partial {count('practice', 'partial')}.")


if __name__ == '__main__':
    check(json.loads(TOPICS.read_text()), json.loads(CAPS.read_text()))
