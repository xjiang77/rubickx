# Git Course — 交互式 Git 基础

属于 Software engineering fundamentals：通过浏览器中的 deterministic simulator 观察和操作 Git 状态。

当前 Gate 1 包含三课：看见 Git 的三区、用 Index 设计 Commit、沿 Commit Graph 移动 HEAD。练习在模拟仓库中执行，不操作本机 Git 仓库。

## 启动与验证

在仓库根目录执行：

```bash
npm --prefix 02-se-fundamentals/git-course ci
make learning-dev
make test-learning-git
make verify-learning-git-gate1
```

`learning-dev` 保留为兼容入口。开发服务器的 `/` 是课程索引，`/git/` 是 Git 课程。完整 gate 包含单元测试、build 与 Playwright；首次运行需安装 Chromium：`cd 02-se-fundamentals/git-course && npx playwright install chromium`。

## 目录

- `src/git/`：Git 状态、命令引擎、课程与 judge。
- `src/components/`：Git 课程界面与仓库状态视图。
- `src/runtime/`：课程 schema、trace 与 progress contract，随当前课程保留。
- `src/catalog-main.tsx`、`git/index.html`：课程索引与 Git 入口。
- `e2e/`：导航、练习完成、进度恢复与手机布局验证。

路径迁移不改变 npm package 名、浏览器 progress key 或现有课程路由。静态首页仍由根目录 `web/` 的既有 GitHub Pages 流程发布。
