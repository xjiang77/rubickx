'use strict';

(async () => {
  const fallback = document.querySelector('#fallback-message');
  try {
    const response = await fetch('./capabilities.json');
    if (!response.ok) throw new Error('目录加载失败');
    const data = await response.json();
    const byId = new Map(data.capabilities.map(c => [c.id, c]));
    // topic 目录与测验清单是增强信息：加载失败时能力图照常可用。
    const optional = async url => { try { const r = await fetch(url); return r.ok ? await r.json() : null; } catch { return null; } };
    const [topicDoc, quizIndex] = await Promise.all([optional('./topics.json'), optional('./quiz/banks/index.json')]);
    const topicsByCap = new Map();
    (topicDoc?.topics || []).forEach(t => { if (!topicsByCap.has(t.capability)) topicsByCap.set(t.capability, []); topicsByCap.get(t.capability).push(t); });
    const quizCaps = new Set((quizIndex?.banks || []).map(b => b.capability));
    const map = document.querySelector('#interactive-map');
    const pillars = document.querySelector('#pillars');
    const capabilities = document.querySelector('#capabilities');
    const extensions = document.querySelector('#extensions');
    const content = document.querySelector('#practice-content');
    const base = 'https://github.com/xjiang77/rubickx/tree/main/';
    let selected;
    let selectedPillar;

    function element(tag, text, className) {
      const el = document.createElement(tag);
      if (text) el.textContent = text;
      if (className) el.className = className;
      return el;
    }
    function link(text, href, className) {
      const el = element('a', text, className);
      el.href = href;
      return el;
    }
    function icon(name) {
      const el = element('img', '', 'icon');
      el.src = `./icons/${name}.svg`;
      el.alt = '';
      el.width = 30;
      el.height = 30;
      return el;
    }
    function node(c, pillar = false) {
      const button = element('button', '', pillar ? `pillar ${c.color}` : 'capability');
      button.type = 'button';
      button.dataset.id = c.id;
      button.setAttribute('aria-pressed', 'false');
      if (pillar) button.append(icon(c.icon));
      const label = element('span', '', 'node-label');
      label.append(element('span', c.title, 'node-title'), element('span', c.zh, 'node-zh'));
      button.append(label);
      button.addEventListener('click', () => {
        const next = pillar ? data.capabilities.find(x => x.pillar === c.id && x.id === data.defaultCapability) || data.capabilities.find(x => x.pillar === c.id) : c;
        select(next.id, true);
      });
      return button;
    }
    function select(id, updateHistory = false) {
      const c = byId.get(id) || byId.get(data.defaultCapability);
      const pillar = data.pillars.find(p => p.id === c.pillar);
      if (selectedPillar !== pillar.id) {
        capabilities.replaceChildren();
        extensions.replaceChildren();
        const group = data.capabilities.filter(x => x.pillar === pillar.id);
        const original = group.filter(x => x.origin === 'andrew');
        const extra = group.filter(x => x.origin === 'rubickx');
        original.forEach(x => capabilities.append(node(x)));
        capabilities.hidden = original.length === 0;
        capabilities.style.setProperty('--count', original.length || 1);
        extensions.hidden = extra.length === 0;
        if (extra.length) {
          extensions.append(element('span', 'Rubickx 扩展', 'extension-label'));
          extra.forEach(x => extensions.append(node(x)));
        }
        selectedPillar = pillar.id;
      }
      map.className = `map ${pillar.color}`;
      map.style.setProperty('--selected-pillar', data.pillars.indexOf(pillar));
      pillars.querySelectorAll('button').forEach(b => b.setAttribute('aria-pressed', String(b.dataset.id === pillar.id)));
      map.querySelectorAll('.capability').forEach(b => b.setAttribute('aria-pressed', String(b.dataset.id === c.id)));
      content.replaceChildren();
      const title = element('h3', `${c.title} / ${c.zh}`);
      title.id = 'practice-title';
      content.append(title, element('p', c.description, 'description'));
      const state = element('p', '', 'state-line');
      state.append(element('span', data.statuses[c.status], `badge ${c.status}`), element('span', c.origin === 'andrew' ? 'Andrew Ng 原框架' : 'Rubickx 扩展'));
      content.append(state);
      if (c.practices.length) {
        const list = element('ul', '', 'practice-list');
        c.practices.forEach(id => {
          const p = data.practices[id];
          const row = element('li');
          row.append(icon(p.icon), element('code', p.id), element('span', p.description, 'practice-description'));
          row.append(link(p.id === 'agent-workflows' ? '工作流说明 ↗' : p.status === 'scaffold' ? '练习骨架 ↗' : '源码 ↗', base + p.path));
          list.append(row);
        });
        content.append(list);
      }
      const topics = topicsByCap.get(c.id) || [];
      if (topics.length) {
        const s = topicDoc.states;
        const head = element('div', '', 'topic-head');
        head.append(element('h4', `Topics · ${topics.length}`), element('span', '知识正文 · ELI5 图解 · 实践', 'topic-legend'));
        if (quizCaps.has(c.id)) head.append(link('能力测验 →', `./quiz/#${c.id}`, 'quiz-link'));
        content.append(head);
        const list = element('ol', '', 'topic-list');
        topics.forEach(t => {
          const row = element('li');
          const chips = element('span', '', 'topic-states');
          chips.append(element('span', '知识 ' + s.knowledge[t.knowledge.state], `chip ${t.knowledge.state}`));
          chips.append(t.eli5 ? link('ELI5 图解', './' + t.eli5.path.replace(/^web\//, ''), 'chip present') : element('span', 'ELI5 ' + s.eli5.missing, 'chip missing'));
          const p = t.practice;
          chips.append(p.state === 'missing' ? element('span', '实践 ' + s.practice.missing, 'chip missing') : link('实践 ' + s.practice[p.state] + ' ↗', base + p.paths[0], `chip ${p.state}`));
          row.append(element('span', t.id, 'topic-id'), element('span', t.title, 'topic-title'), chips);
          list.append(row);
        });
        content.append(list);
      }
      content.append(element('p', c.boundary, 'boundary'));
      if (c.status === 'planned') content.append(element('p', '下一步：' + c.next, 'next-step'));
      const links = element('div', '', 'capability-links');
      links.append(link('能力说明 README ↗', base + c.path + '/README.md'));
      c.related.forEach(id => {
        const other = byId.get(id);
        const a = link(other.zh, '#' + id);
        a.dataset.capability = id;
        links.append(a);
      });
      content.append(links);
      document.querySelector('#selection-status').textContent = `${pillar.zh}，${c.zh}，${data.statuses[c.status]}`;
      if (selected && selected !== c.id && !matchMedia('(prefers-reduced-motion: reduce)').matches) content.animate([{ opacity: 0.35 }, { opacity: 1 }], { duration: 200 });
      selected = c.id;
      if (updateHistory && location.hash !== '#' + c.id) history.pushState(null, '', '#' + c.id);
    }

    data.pillars.forEach(p => pillars.append(node(p, true)));
    select(location.hash.slice(1));
    map.hidden = false;
    fallback.hidden = true;
    document.querySelector('#all-capabilities').open = false;
    document.querySelector('#all-practices').open = false;
    function followHash() {
      const id = location.hash.slice(1);
      if (!id || byId.has(id)) select(id);
      if (id === 'all-practices' || id === 'all-capabilities') document.getElementById(id).open = true;
    }
    followHash();
    window.addEventListener('hashchange', followHash);
    window.addEventListener('popstate', followHash);
    document.addEventListener('click', event => {
      const a = event.target.closest('a[data-capability]');
      if (a && byId.has(a.dataset.capability)) {
        event.preventDefault();
        select(a.dataset.capability, true);
        map.querySelector(`.capability[data-id="${a.dataset.capability}"]`).focus();
      }
      // 重复点击当前锚点不触发 hashchange，也应能重新打开目录。
      const browse = event.target.closest('a[href="#all-practices"], a[href="#all-capabilities"]');
      if (browse) document.querySelector(browse.getAttribute('href')).open = true;
    });
  } catch (error) {
    document.querySelector('#interactive-map').hidden = true;
    fallback.hidden = false;
    fallback.textContent = '交互图暂不可用。仍可从下方完整目录查看全部能力说明与实践入口。';
    document.querySelectorAll('.directory').forEach(d => { d.open = true; });
  }
})();
