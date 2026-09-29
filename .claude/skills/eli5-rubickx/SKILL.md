---
name: eli5-rubickx
description: 为 Rubickx Skills Map 的一个 topic 写 ELI5 页（大图少字的自包含 HTML），从私有知识库正文派生，登记到 web/topics.json 并通过 check_eli5 / check_topics。用于"给 topic 1.2.3 做 ELI5""补这个能力的图解页"。
---

# eli5-rubickx

以社区 [eli5 skill](https://github.com/anthropics/claude-plugins-community/blob/main/eli5/skills/eli5/SKILL.md) 为基底：**读者对这个主题一无所知，用 HTML、大图、少字来讲。** 本文件在它之上加 Rubickx 的页面合同；合同写在仓库里，`scripts/check_eli5.py` 才有据可查。

## 输入

一个 topic id（如 `1.6.1`）。从 `web/topics.json` 读出：能力、标题、`knowledge.notes`（私有知识库 dragon-vault 里的正文路径）、`practice`。

## 步骤

1. **读正文**。打开 `knowledge.notes` 里的每一篇（在 dragon-vault 中）。机制与判断只从正文取，不另加正文里没有的结论。正文未核验时，页面顶部加 `<p class="flag">样板页：对应的知识正文尚待人工核验…</p>`。
2. **先给背景（「先弄清楚」三块，缺一不可）**。例子里的每个变量都要有实际含义（输入、参数、预测、正确答案、误差、损失），不用只有符号的纯数学式。零基础的读者先要知道为什么看这页，才看得懂后面的图：
   - `.ctx.why`「为什么要懂它」：它解决什么问题，不懂它会卡在哪里；
   - `.ctx.what`「它是什么」：定义要结合后面要用的具体例子来讲，配一张图，让读者在例子上看到定义说的是什么；
   - `.ctx.origin`「从哪里来」：历史或来由，能画成时间线就画。正文没有写历史时，可以用公开来源补，但来源要列在「想深入」里，并在知识库正文补上对应一节。
   三块文字合计不超过 450 个汉字。
3. **再用一个例子推演一遍（3–6 屏）**。先用 `.intro` 说明例子是什么、和真实情况是什么关系，并用列表写出**这个例子要说明的直觉**（通常 2–3 条），后面每一屏都服务于其中一条。涉及计算或机制的 topic，每屏给出对应的公式（放在 `.formula` 里），数值要能逐步核对，能做数值验证的就写出验证。每屏一张内联 SVG 主图，配一两句话；图里的每个元素都要有标注（例如"输入""第 1 步""损失"），不让读者猜它代表什么。所有屏的说明文字合计不超过 400 个汉字；术语第一次出现时用一句话解释。一页只讲一个核心直觉，讲全是正文的事。
4. **画图**（页面结构从上到下：标题与一句话导语 → 先弄清楚 → 小例子 → 固定结尾）。
   - 复制 `web/learn/_template.html`，**不要改 `<style id="eli5-base">`**，检查脚本会逐字比对。`body` 的 class 按 pillar 取 `ai` / `se` / `agents` / `build`。
   - 用模板里的 SVG 类：`box`、`hl`、`op`、`fwd`、`bwd`、`faint`、`t`、`tb`、`tc`、`ts`。箭头 marker 放在页首隐藏的 `<svg><defs>` 里，填色写 pillar 色的字面值。
   - 每张 SVG 带 `role="img"` 与说清画面内容的 `aria-label`。
   - 手机上 800 宽的 viewBox 大约缩到 0.44 倍：图内文字至少用 `ts`（22），能放进段落的解释就别写在图里。按内容裁 viewBox，不留大片空白。
   - 不用 emoji、外部字体、外部脚本或图片。
   - 前置概念第一次出现时，用 `<a class="term">` 链接到对应的图解页锚点（例如 `math-prerequisites.html#chain-rule`），读者需要时一点就能补上。
5. **固定结尾**，按顺序六块都要有：
   - `.limits`「适用边界」：这个例子说明不了什么、在什么条件下结论不成立；用了类比时，写出类比在哪里失效；
   - `.uses`「在 AI 工程里什么时候用得上」：3–4 个真实场景，每个写清怎样用这个概念做判断（例如 loss 变成 NaN 时先看什么）；
   - `.summary`：一句话总结；
   - `.tree`「知识树」：前置、本体、延展、用在哪里四个方向，按 `web/topics.json` 的 `relations` 与知识库正文的「知识树」一节填写；有 ELI5 页的链接过去，没有的写 topic 编号并标“图解待写”；
   - `.try`「去试试」：实践路径与命令；实践待建时写"实践待建"和计划；
   - `.deeper`「想深入」：正文标题（知识库不公开，只写标题，不放链接）和公开来源链接。
6. **元数据**：`rubickx-topic`；`rubickx-derived-from` 写成"正文路径@正文 frontmatter 的 updated"，供知识库侧检查页面是否过期。
7. **登记**：文件放在 `web/learn/<capability-id>/<slug>.html`；在 `web/topics.json` 的该 topic 下写 `eli5: {path, derived_from: {note, updated}}`。
8. **检查**：`python3 scripts/check_eli5.py && python3 scripts/check_topics.py && python3 scripts/render_skills_map.py --check`。再在 1280 宽和 390 宽下各截一张图看一遍：无横向滚动，图内文字能读。

## 用词

按 dragon-vault 的 `99_System/Specs/02_Chinese_Style_Spec.md` 与 `03_Chinese_Style_Examples.md`：

- 标题直接写概念名与它回答的问题（如“Backpropagation（反向传播）：沿 computational graph 求出每个 parameter 的 gradient”），不用悬念式标题。
- **专业术语用英文原词**，第一次出现时在括号里给常用中文译名，之后只用英文：例如 loss function（损失函数）、gradient（梯度）、learning rate（学习率）、computational graph（计算图）。句首的英文术语首字母大写。日常词语（输入、预测、误差、正确答案）用中文。
- 第一次出现时先给直观含义，再给术语：例如“每个节点是一个中间结果，每条箭头是一次运算，这张图就是 computational graph”。同一概念全文称谓一致，不自造简称或口语化的替代说法（例如不用“放大倍数”代替 local derivative）。
- 用动词写清谁对什么做了什么；比较与程度要说明对象和尺度；依据不足时收窄结论。
- 术语与正文保持一致：页面用词跟随知识库正文，例如可行性三类用正文的“不可能 / 代价问题 / 可靠性问题”。

## 不能做的事

- **不给练习答案**。实践是本人手写的练习时（例如 nanochat 的 C5 约束），页面只讲直觉和练习入口，不贴实现代码。
- **不越过公开边界**。页面随 GitHub Pages 公开：不写个人项目的内部事实、内部数字或私有链接；拿不准时先问。
- **不替正文下结论**。正文标为"未验证"的判断，页面也不能说成定论。
