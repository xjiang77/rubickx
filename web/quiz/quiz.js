'use strict';

// 能力测验：题库来自 banks/*.json；作答历史只存在本机 localStorage，可导出导入。
(async () => {
  const KEY = 'rubickx.quiz.v1';
  const PER_ATTEMPT = 10;
  const $ = s => document.querySelector(s);
  const el = (tag, text, cls) => { const e = document.createElement(tag); if (text != null) e.textContent = text; if (cls) e.className = cls; return e; };
  const notice = msg => { const n = $('#notice'); n.textContent = msg; n.hidden = false; };

  // ---- 存储：localStorage 不可用时退回内存，并提示用户及时导出 ----
  let memory = { attempts: [] };
  let persistent = true;
  function load() {
    try { const raw = localStorage.getItem(KEY); if (raw) memory = JSON.parse(raw); } catch { persistent = false; }
    if (!Array.isArray(memory.attempts)) memory = { attempts: [] };
    if (!persistent) notice('这个浏览器不允许保存记录：本次成绩只在页面打开期间有效，离开前请导出。');
  }
  function save() {
    if (!persistent) return;
    try { localStorage.setItem(KEY, JSON.stringify(memory)); } catch { persistent = false; notice('保存失败：本次成绩只在页面打开期间有效，离开前请导出。'); }
  }

  // ---- 数据 ----
  const getJSON = async url => { const r = await fetch(url); if (!r.ok) throw new Error(url); return r.json(); };
  let caps, index, topics;
  try {
    [caps, index, topics] = await Promise.all([getJSON('../capabilities.json'), getJSON('./banks/index.json'), getJSON('../topics.json').catch(() => ({ topics: [] }))]);
  } catch {
    notice('题库加载失败，请稍后再试。');
    return;
  }
  load();
  const capById = new Map(caps.capabilities.map(c => [c.id, c]));
  const topicById = new Map(topics.topics.map(t => [t.id, t]));
  const banks = new Map();
  const base = 'https://github.com/xjiang77/rubickx/tree/main/';

  // ---- 选择能力 ----
  const select = $('#capability');
  index.banks.forEach(b => {
    const c = capById.get(b.capability);
    if (!c) return;
    const opt = el('option', `${c.title} / ${c.zh}`);
    opt.value = b.capability;
    select.append(opt);
  });
  if (!select.options.length) { notice('还没有可用的题库。'); return; }
  const fromHash = decodeURIComponent(location.hash.slice(1));
  if (index.banks.some(b => b.capability === fromHash)) select.value = fromHash;
  $('#picker').hidden = false;

  async function bank(capId) {
    if (!banks.has(capId)) {
      const entry = index.banks.find(b => b.capability === capId);
      banks.set(capId, await getJSON('./' + entry.path));
    }
    return banks.get(capId);
  }

  async function showPicker() {
    const capId = select.value;
    if (location.hash !== '#' + capId) history.replaceState(null, '', '#' + capId);
    const b = await bank(capId);
    const topicIds = [...new Set(b.questions.map(q => q.topic))];
    const n = Math.min(PER_ATTEMPT, b.questions.length);
    $('#bank-meta').textContent = `题库 v${b.version}${b.status === 'sample' ? '（样板）' : ''} · ${b.questions.length} 道题，覆盖 topic ${topicIds.join('、')} · 每次抽 ${n} 道`;
    $('#to-map').href = '../index.html#' + capId;
    renderHistory(capId, b);
  }
  select.addEventListener('change', () => { $('#result').hidden = true; showPicker(); });

  // ---- 抽题与作答 ----
  const shuffle = a => { const x = a.slice(); for (let i = x.length - 1; i > 0; i--) { const j = Math.floor(Math.random() * (i + 1)); [x[i], x[j]] = [x[j], x[i]]; } return x; };
  let run = null;

  $('#start').addEventListener('click', async () => {
    const capId = select.value;
    const b = await bank(capId);
    const picked = shuffle(b.questions).slice(0, Math.min(PER_ATTEMPT, b.questions.length));
    run = {
      capId, version: b.version, startedAt: new Date().toISOString(), i: 0, items: [],
      questions: picked.map(q => ({ ...q, order: shuffle(q.options.map((_, k) => k)) })),
    };
    $('#picker').hidden = true; $('#result').hidden = true; $('#history').hidden = true;
    $('#question').hidden = false;
    showQuestion();
  });

  function chosen() {
    return [...document.querySelectorAll('#q-options input:checked')].map(i => Number(i.value));
  }

  function showQuestion() {
    const q = run.questions[run.i];
    $('#progress').textContent = `第 ${run.i + 1} / ${run.questions.length} 题`;
    const t = topicById.get(q.topic);
    $('#q-topic').textContent = t ? `${q.topic} ${t.title}` : q.topic;
    $('#q-kind').textContent = q.scenario ? '场景题' : (q.type === 'multi' ? '多选' : '单选');
    $('#q-kind').hidden = false;
    $('#q-prompt').textContent = q.prompt;
    $('#q-hint').textContent = q.type === 'multi' ? '可多选；全部选对才算对。' : '';
    const box = $('#q-options');
    box.querySelectorAll('.opt').forEach(n => n.remove());
    q.order.forEach(k => {
      const lab = el('label', null, 'opt');
      const input = el('input');
      input.type = q.type === 'multi' ? 'checkbox' : 'radio';
      input.name = 'opt';
      input.value = String(k);
      input.addEventListener('change', () => { $('#submit').disabled = chosen().length === 0; });
      lab.append(input, el('span', q.options[k]));
      box.append(lab);
    });
    $('#feedback').hidden = true;
    $('#submit').hidden = false; $('#submit').disabled = true;
    $('#next').hidden = true;
  }

  $('#submit').addEventListener('click', () => {
    const q = run.questions[run.i];
    const pick = chosen().sort();
    const answer = q.answer.slice().sort();
    const ok = pick.length === answer.length && pick.every((v, k) => v === answer[k]);
    run.items.push({ qid: q.id, topic: q.topic, correct: ok });
    document.querySelectorAll('#q-options .opt').forEach(lab => {
      const input = lab.querySelector('input');
      const k = Number(input.value);
      input.disabled = true;
      if (q.answer.includes(k)) lab.classList.add('correct');
      else if (input.checked) lab.classList.add('wrong');
    });
    const fb = $('#feedback');
    fb.replaceChildren();
    fb.className = 'feedback ' + (ok ? 'ok' : 'bad');
    fb.append(el('strong', ok ? '答对了' : '没答对，正确答案已标绿'), el('span', q.explain));
    const t = topicById.get(q.topic);
    if (t) {
      const links = el('div', null, 'links');
      if (t.eli5) { const a = el('a', '看 ELI5 图解'); a.href = '../' + t.eli5.path.replace(/^web\//, ''); links.append(a); }
      if (t.practice.state !== 'missing') { const a = el('a', '去练习 ↗'); a.href = base + t.practice.paths[0]; links.append(a); }
      if (links.childElementCount) fb.append(links);
    }
    fb.hidden = false;
    $('#submit').hidden = true;
    $('#next').hidden = false;
    $('#next').textContent = run.i + 1 < run.questions.length ? '下一题' : '看结果';
    $('#next').focus();
  });

  $('#next').addEventListener('click', () => {
    run.i += 1;
    if (run.i < run.questions.length) { showQuestion(); return; }
    finish();
  });

  function finish() {
    const correct = run.items.filter(x => x.correct).length;
    const attempt = {
      id: `${run.capId}@${run.startedAt}`,
      capability: run.capId, version: run.version,
      startedAt: run.startedAt, finishedAt: new Date().toISOString(),
      total: run.items.length, correct, items: run.items,
    };
    memory.attempts.push(attempt);
    save();
    $('#question').hidden = true;
    $('#score').textContent = `${correct} / ${attempt.total}（${Math.round(100 * correct / attempt.total)}%）`;
    const box = $('#result-topics');
    box.replaceChildren();
    topicBars(box, [attempt]);
    $('#result').hidden = false;
    $('#picker').hidden = false;
    run = null;
    showPicker().then(() => { $('#history').hidden = false; });
  }

  $('#again').addEventListener('click', () => { $('#result').hidden = true; $('#picker').hidden = false; $('#start').click(); });

  // ---- 历史 ----
  function topicBars(box, attempts) {
    const agg = new Map();
    attempts.forEach(a => a.items.forEach(it => {
      const v = agg.get(it.topic) || { ok: 0, n: 0 };
      v.n += 1; if (it.correct) v.ok += 1;
      agg.set(it.topic, v);
    }));
    [...agg.entries()].sort().forEach(([id, v]) => {
      const pct = Math.round(100 * v.ok / v.n);
      const row = el('div', null, 'bar');
      const t = topicById.get(id);
      row.title = t ? t.title : id;
      const track = el('div', null, 'track');
      const fill = el('div', null, 'fill' + (pct < 60 ? ' low' : ''));
      fill.style.width = pct + '%';
      track.append(fill);
      row.append(el('span', id), track, el('span', `${v.ok}/${v.n} · ${pct}%`));
      box.append(row);
    });
  }

  function trend(attempts) {
    const W = 600, H = 120, P = 28;
    const ns = 'http://www.w3.org/2000/svg';
    const svg = document.createElementNS(ns, 'svg');
    svg.setAttribute('viewBox', `0 0 ${W} ${H}`);
    svg.setAttribute('class', 'trend');
    svg.setAttribute('role', 'img');
    svg.setAttribute('aria-label', '历次得分：' + attempts.map(a => Math.round(100 * a.correct / a.total) + '%').join('，'));
    const xs = i => attempts.length === 1 ? W / 2 : P + i * (W - 2 * P) / (attempts.length - 1);
    const ys = a => H - P - (H - 2 * P) * a.correct / a.total;
    const add = (tag, attrs) => { const n = document.createElementNS(ns, tag); Object.entries(attrs).forEach(([k, v]) => n.setAttribute(k, v)); svg.append(n); return n; };
    add('line', { x1: P, y1: H - P, x2: W - P, y2: H - P, stroke: '#d9dedb' });
    add('line', { x1: P, y1: P, x2: W - P, y2: P, stroke: '#eceeed', 'stroke-dasharray': '4 4' });
    const label = (y, s) => { const n = add('text', { x: 0, y: y + 4, 'font-size': 11, fill: '#66706c' }); n.textContent = s; };
    label(P, '100%'); label(H - P, '0%');
    if (attempts.length > 1) add('polyline', { points: attempts.map((a, i) => `${xs(i)},${ys(a)}`).join(' '), fill: 'none', stroke: '#246b35', 'stroke-width': 2.5 });
    attempts.forEach((a, i) => add('circle', { cx: xs(i), cy: ys(a), r: 4.5, fill: '#246b35' }));
    return svg;
  }

  function renderHistory(capId, b) {
    const body = $('#history-body');
    body.replaceChildren();
    const mine = memory.attempts.filter(a => a.capability === capId).sort((x, y) => x.startedAt.localeCompare(y.startedAt));
    $('#history').hidden = false;
    if (!mine.length) { body.append(el('p', '还没有这个能力的作答记录。', 'meta')); return; }
    const current = mine.filter(a => a.version === b.version);
    if (current.length) {
      body.append(el('p', `题库 v${b.version} 的历次得分`, 'meta'), trend(current));
      body.append(el('p', `各 topic 累计正确率（题库 v${b.version}，${current.length} 次）`, 'meta'));
      topicBars(body, current);
    }
    const table = el('table');
    const head = el('tr');
    ['时间', '题库', '得分'].forEach(h => head.append(el('th', h)));
    table.append(head);
    mine.slice().reverse().forEach(a => {
      const tr = el('tr');
      const d = new Date(a.finishedAt);
      tr.append(el('td', d.toLocaleString('zh-CN', { hour12: false })), el('td', 'v' + a.version), el('td', `${a.correct}/${a.total}（${Math.round(100 * a.correct / a.total)}%）`));
      table.append(tr);
    });
    body.append(el('p', '全部记录（不同题库版本分开计分）', 'meta'), table);
  }

  // ---- 导出与导入 ----
  $('#export').addEventListener('click', () => {
    const blob = new Blob([JSON.stringify({ format: KEY, exportedAt: new Date().toISOString(), ...memory }, null, 2)], { type: 'application/json' });
    const a = el('a');
    a.href = URL.createObjectURL(blob);
    a.download = `rubickx-quiz-history-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.append(a); a.click(); a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 1000);
  });
  $('#import').addEventListener('change', async e => {
    const file = e.target.files[0];
    if (!file) return;
    try {
      const data = JSON.parse(await file.text());
      if (!Array.isArray(data.attempts)) throw new Error('format');
      const seen = new Set(memory.attempts.map(a => a.id));
      let added = 0;
      data.attempts.forEach(a => {
        if (a && a.id && a.capability && Array.isArray(a.items) && !seen.has(a.id)) { memory.attempts.push(a); seen.add(a.id); added += 1; }
      });
      save();
      notice(`已导入 ${added} 条记录（跳过 ${data.attempts.length - added} 条重复或无效记录）。`);
      showPicker();
    } catch {
      notice('导入失败：文件不是有效的测验历史 JSON。');
    }
    e.target.value = '';
  });

  window.addEventListener('hashchange', () => {
    const id = decodeURIComponent(location.hash.slice(1));
    if (index.banks.some(b => b.capability === id)) { select.value = id; showPicker(); }
  });
  showPicker();
})();
