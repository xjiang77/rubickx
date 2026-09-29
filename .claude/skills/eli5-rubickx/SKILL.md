---
name: eli5-rubickx
description: 为 Rubickx Skills Map 的一个 topic 写 ELI5 页（大图少字的自包含 HTML），从私有知识库正文派生，登记到 web/topics.json 并通过 check_eli5 / check_topics。用于"给 topic 1.2.3 做 ELI5""补这个能力的图解页"。
---

# eli5-rubickx

以社区 [eli5 skill](https://github.com/anthropics/claude-plugins-community/blob/main/eli5/skills/eli5/SKILL.md) 为基底：**读者对这个主题一无所知，用 HTML、大图、少字来讲。** 本文件在它之上加 Rubickx 的页面合同；合同写在仓库里，`scripts/check_eli5.py` 才有据可查。

## 输入

一个 topic id（如 `1.6.1`）。从 `web/topics.json` 读出：能力、标题、`knowledge.notes`（私有知识库 dragon-vault 里的正文路径）、`practice`。

## 步骤

1. **读正文**。打开 `knowledge.notes` 里的每一篇（在 dragon-vault 中）。只从正文取内容，不另加正文里没有的结论。正文未核验时，页面顶部加 `<p class="flag">样板页：对应的知识正文尚待人工核验…</p>`。
2. **挑一个核心直觉**。一页只讲一个：读者看完能在脑子里留下的那张图。讲全是正文的事。
3. **分 3–6 屏**。每屏一张内联 SVG 主图，配一两句话。所有屏的说明文字合计不超过 400 个汉字；术语第一次出现时用一句话解释。
4. **画图**。
   - 复制 `web/learn/_template.html`，**不要改 `<style id="eli5-base">`**，检查脚本会逐字比对。`body` 的 class 按 pillar 取 `ai` / `se` / `agents` / `build`。
   - 用模板里的 SVG 类：`box`、`hl`、`op`、`fwd`、`bwd`、`faint`、`t`、`tb`、`tc`、`ts`。箭头 marker 放在页首隐藏的 `<svg><defs>` 里，填色写 pillar 色的字面值。
   - 每张 SVG 带 `role="img"` 与说清画面内容的 `aria-label`。
   - 手机上 800 宽的 viewBox 大约缩到 0.44 倍：图内文字至少用 `ts`（22），能放进段落的解释就别写在图里。按内容裁 viewBox，不留大片空白。
   - 不用 emoji、外部字体、外部脚本或图片。
5. **固定结尾**，四块都要有：
   - `.limits`「这个比喻在哪里不成立」：页里用了类比，就写出它在哪里失效；
   - `.summary`：一句话总结；
   - `.try`「去试试」：实践路径与命令；实践待建时写"实践待建"和计划；
   - `.deeper`「想深入」：正文标题（知识库不公开，只写标题，不放链接）和公开来源链接。
6. **元数据**：`rubickx-topic`；`rubickx-derived-from` 写成"正文路径@正文 frontmatter 的 updated"，供知识库侧检查页面是否过期。
7. **登记**：文件放在 `web/learn/<capability-id>/<slug>.html`；在 `web/topics.json` 的该 topic 下写 `eli5: {path, derived_from: {note, updated}}`。
8. **检查**：`python3 scripts/check_eli5.py && python3 scripts/check_topics.py && python3 scripts/render_skills_map.py --check`。再在 1280 宽和 390 宽下各截一张图看一遍：无横向滚动，图内文字能读。

## 不能做的事

- **不给练习答案**。实践是本人手写的练习时（例如 nanochat 的 C5 约束），页面只讲直觉和练习入口，不贴实现代码。
- **不越过公开边界**。页面随 GitHub Pages 公开：不写个人项目的内部事实、内部数字或私有链接；拿不准时先问。
- **不替正文下结论**。正文标为"未验证"的判断，页面也不能说成定论。
