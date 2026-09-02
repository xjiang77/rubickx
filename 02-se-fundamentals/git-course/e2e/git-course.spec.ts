import { expect, test } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("/git/");
  await page.evaluate(() => localStorage.clear());
  await page.reload();
});

test("catalog enters the Git course with local assets", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Interactive Git Course" })).toBeVisible();
  await page.getByRole("link", { name: /Interactive Git Course/ }).click();
  await expect(page).toHaveURL(/\/git\/$/);
  await expect(page.getByRole("heading", { name: "看见 Git 的三区" })).toBeVisible();
});

test("completes all released reference solutions and keeps progress after reload", async ({ page }) => {
  const cases = [
    { lesson: "看见 Git 的三区", command: "git add app.ts" },
    {
      lesson: "用 Index 设计 Commit",
      command: 'git add app.ts; git commit -m "code"; git add README.md; git commit -m "docs"',
    },
    { lesson: "沿 Commit Graph 移动 HEAD", command: "git switch --detach HEAD^" },
  ];

  for (const item of cases) {
    await page.getByRole("button", { name: new RegExp(item.lesson) }).click();
    const input = page.getByLabel("$");
    await input.fill(item.command);
    await input.press("Enter");
    await expect(page.getByRole("button", { name: "Judge my state" })).toBeEnabled();
    await page.getByRole("button", { name: "Judge my state" }).click();
    await expect(page.getByRole("status")).toContainText("目标达成");
  }

  await expect(page.getByText("3 / 3 lessons")).toBeVisible();
  await page.reload();
  await expect(page.getByText("3 / 3 lessons")).toBeVisible();
  await expect(page.getByRole("heading", { name: "沿 Commit Graph 移动 HEAD" })).toBeVisible();
});

test("supports a mobile stacked layout without horizontal page overflow", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/git/");
  await expect(page.getByRole("heading", { name: "看见 Git 的三区" })).toBeVisible();
  const widths = await page.evaluate(() => ({ scroll: document.documentElement.scrollWidth, client: document.documentElement.clientWidth }));
  expect(widths.scroll).toBeLessThanOrEqual(widths.client + 1);
});
