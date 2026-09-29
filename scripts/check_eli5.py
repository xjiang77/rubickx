#!/usr/bin/env python3
"""校验 web/learn/ 下的 ELI5 页满足页面合同（skills/eli5-rubickx/SKILL.md）。

检查能机器判定的部分：自包含、基础样式与模板一致、屏数与图、文字预算、固定结尾、
回链与元数据。讲得好不好、比喻贴不贴切，仍由人过目。
"""
import re
import sys
from html.parser import HTMLParser
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
LEARN = ROOT / 'web/learn'
TEMPLATE = LEARN / '_template.html'
BASE = re.compile(r'<style id="eli5-base">.*?</style>', re.S)
CJK = re.compile(r'[一-鿿]')
SCREEN_TEXT_BUDGET = 400   # 所有屏的说明文字合计汉字数上限
CONTEXT_TEXT_BUDGET = 450  # 「先弄清楚」三块文字合计汉字数上限
MIN_SCREENS, MAX_SCREENS = 3, 6


class Page(HTMLParser):
    def __init__(self):
        super().__init__()
        self.stack = []
        self.screens = 0
        self.figures = 0
        self.labelled_svgs = 0
        self.screen_text = []
        self.context_text = []
        self.ctx_blocks = set()
        self.external = []
        self.scripts_with_src = 0
        self.blocks = set()

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        cls = set((a.get('class') or '').split())
        if tag not in ('br', 'img', 'meta', 'link', 'input', 'path', 'line', 'rect', 'circle', 'ellipse', 'polyline', 'polygon', 'use'):
            self.stack.append((tag, cls))
        if tag == 'section' and 'screen' in cls:
            self.screens += 1
        if tag == 'figure' and self.in_class('screen'):
            self.figures += 1
        if tag == 'svg' and self.in_class('screen') and a.get('role') == 'img' and a.get('aria-label'):
            self.labelled_svgs += 1
        if tag == 'div' and 'ctx' in cls:
            self.ctx_blocks |= cls & {'why', 'what', 'origin'}
        for c in ('limits', 'summary', 'try', 'deeper'):
            if c in cls:
                self.blocks.add(c)
        if tag == 'script' and a.get('src'):
            self.scripts_with_src += 1
        for key in ('src', 'href'):
            v = a.get(key) or ''
            if tag in ('link', 'script', 'img', 'image', 'source', 'iframe') and re.match(r'^(https?:)?//', v):
                self.external.append(v)

    def handle_endtag(self, tag):
        for i in range(len(self.stack) - 1, -1, -1):
            if self.stack[i][0] == tag:
                del self.stack[i:]
                break

    def handle_data(self, data):
        if self.in_class('screen') and not any(t == 'svg' for t, _ in self.stack) and any(t == 'p' for t, _ in self.stack):
            self.screen_text.append(data)
        if self.in_class('ctx') and not any(t == 'svg' for t, _ in self.stack) and any(t == 'p' for t, _ in self.stack):
            self.context_text.append(data)

    def in_class(self, name):
        return any(name in cls for _, cls in self.stack)


def check_page(path, base):
    rel = path.relative_to(ROOT)
    html = path.read_text()
    errors = []
    m = BASE.search(html)
    if not m or m.group(0) != base:
        errors.append('基础样式与 _template.html 不一致')
    for pattern, what in [
        (r'^<!DOCTYPE html>', 'doctype'),
        (r'<html lang="zh-CN">', 'lang'),
        (r'<meta name="viewport"', 'viewport'),
        (r'<meta name="rubickx-topic" content="[1-4]\.\d\.\d">', 'topic 元数据'),
        (r'<meta name="rubickx-derived-from" content="02_Knowledge/[^"]+\.md@\d{4}-\d{2}-\d{2}">', '派生来源元数据'),
        (r'<title>[^<]+ · ELI5 · Rubickx</title>', 'title'),
    ]:
        if not re.search(pattern, html):
            errors.append(f'缺 {what}')
    if not re.search(r'<div class="intro">.*?<ol>\s*<li>.*?</ol>.*?</div>', html, re.S):
        errors.append('例子前缺 .intro：要说明例子是什么，并列出这个例子要说明的直觉')
    cap = path.parent.name
    if f'href="../../index.html#{cap}"' not in html:
        errors.append('缺回到能力地图的链接')
    p = Page()
    p.feed(html)
    if not MIN_SCREENS <= p.screens <= MAX_SCREENS:
        errors.append(f'屏数 {p.screens} 不在 {MIN_SCREENS}–{MAX_SCREENS}')
    if p.figures < p.screens or p.labelled_svgs < p.screens:
        errors.append('每屏需要一张带 role="img" 与 aria-label 的 SVG 图')
    n = len(CJK.findall(''.join(p.screen_text)))
    if n > SCREEN_TEXT_BUDGET:
        errors.append(f'屏内说明 {n} 字，超过 {SCREEN_TEXT_BUDGET}')
    lacking = {'why', 'what', 'origin'} - p.ctx_blocks
    if lacking:
        errors.append(f'「先弄清楚」缺 {sorted(lacking)}（为什么要懂它 / 它是什么 / 从哪里来）')
    m = len(CJK.findall(''.join(p.context_text)))
    if m > CONTEXT_TEXT_BUDGET:
        errors.append(f'「先弄清楚」{m} 字，超过 {CONTEXT_TEXT_BUDGET}')
    missing = {'limits', 'summary', 'try', 'deeper'} - p.blocks
    if missing:
        errors.append(f'缺固定结尾块 {sorted(missing)}')
    if p.external or p.scripts_with_src:
        errors.append(f'不能加载外部资源 {p.external}')
    return rel, (n, m), errors


def main():
    base = BASE.search(TEMPLATE.read_text()).group(0)
    pages = sorted(p for p in LEARN.rglob('*.html') if not p.name.startswith('_'))
    failed = False
    for page in pages:
        rel, n, errors = check_page(page, base)
        if errors:
            failed = True
            print(f'FAIL {rel}: ' + '；'.join(errors))
        else:
            print(f'ok   {rel}（先弄清楚 {n[1]} 字，屏内 {n[0]} 字）')
    if failed:
        sys.exit(1)
    print(f'ELI5 gate passed: {len(pages)} pages.')


if __name__ == '__main__':
    main()
