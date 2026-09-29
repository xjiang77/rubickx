#!/usr/bin/env python3
"""校验能力测验题库（web/quiz/banks/）：结构、答案合法、topic 归属与覆盖、场景题比例。

题目好不好、测的是不是理解，要靠人试做；这里只拦下机器能判定的错误。
完整题库（status: complete）要求覆盖能力下全部 topic；样板题库（status: sample）
只要求所含 topic 各有足够题目。
"""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
QUIZ = ROOT / 'web/quiz'
MIN_PER_TOPIC = 3
MIN_TOTAL = 12
MIN_SCENARIO_RATIO = 0.25


def check_bank(path, caps, topics):
    b = json.loads(path.read_text())
    where = path.name
    assert b['capability'] in caps, where
    assert path.stem == b['capability'], (where, '文件名须与能力 id 一致')
    assert isinstance(b['version'], int) and b['version'] >= 1, where
    assert b['status'] in ('sample', 'complete'), where
    qs = b['questions']
    ids = [q['id'] for q in qs]
    assert len(ids) == len(set(ids)), (where, '题目 id 重复')
    cap_topics = {t['id'] for t in topics if t['capability'] == b['capability']}
    per_topic = {}
    for q in qs:
        w = (where, q['id'])
        assert q['topic'] in cap_topics, (w, 'topic 不属于这个能力')
        assert q['type'] in ('single', 'multi'), w
        assert q['prompt'].strip() and q['explain'].strip(), w
        opts = q['options']
        assert len(opts) >= 3 and len(set(opts)) == len(opts), (w, '至少 3 个互不相同的选项')
        ans = q['answer']
        assert ans and len(set(ans)) == len(ans) and all(0 <= a < len(opts) for a in ans), (w, '答案下标非法')
        if q['type'] == 'single':
            assert len(ans) == 1, (w, '单选只能有一个答案')
        else:
            assert len(ans) < len(opts), (w, '多选不能全选')
        assert isinstance(q['scenario'], bool), w
        per_topic[q['topic']] = per_topic.get(q['topic'], 0) + 1
    thin = {t: n for t, n in per_topic.items() if n < MIN_PER_TOPIC}
    assert not thin, (where, f'每个 topic 至少 {MIN_PER_TOPIC} 题', thin)
    assert len(qs) >= MIN_TOTAL, (where, f'至少 {MIN_TOTAL} 题')
    ratio = sum(q['scenario'] for q in qs) / len(qs)
    assert ratio >= MIN_SCENARIO_RATIO, (where, f'场景题比例 {ratio:.0%} 低于 {MIN_SCENARIO_RATIO:.0%}')
    if b['status'] == 'complete':
        missing = cap_topics - set(per_topic)
        assert not missing, (where, '完整题库须覆盖全部 topic', sorted(missing))
    return b, len(qs), ratio


def main():
    caps = {c['id'] for c in json.loads((ROOT / 'web/capabilities.json').read_text())['capabilities']}
    topics = json.loads((ROOT / 'web/topics.json').read_text())['topics']
    index = json.loads((QUIZ / 'banks/index.json').read_text())
    listed = {e['path'] for e in index['banks']}
    on_disk = {f'banks/{p.name}' for p in (QUIZ / 'banks').glob('*.json') if p.name != 'index.json'}
    assert listed == on_disk, ('index.json 与题库文件不一致', sorted(listed ^ on_disk))
    for e in index['banks']:
        b, n, ratio = check_bank(QUIZ / e['path'], caps, topics)
        assert b['capability'] == e['capability'], e
        print(f"ok   {e['path']}：v{b['version']} {b['status']}，{n} 题，场景题 {ratio:.0%}")
    print(f'Quiz gate passed: {len(index["banks"])} bank(s).')


if __name__ == '__main__':
    main()
