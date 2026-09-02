import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GitCourseApp } from "./GitCourseApp";

describe("GitCourseApp", () => {
  beforeEach(() => {
    localStorage.clear();
    window.scrollTo = vi.fn();
    window.matchMedia = vi.fn().mockReturnValue({ matches: true });
  });

  it("renders the complete learning loop and Gate 1 navigation", () => {
    render(<GitCourseApp />);
    expect(screen.getByRole("heading", { name: "看见 Git 的三区" })).toBeInTheDocument();
    expect(screen.getByLabelText("Learning loop")).toHaveTextContent("Explain");
    expect(screen.getByLabelText("Learning loop")).toHaveTextContent("Checkpoint");
    expect(screen.getByRole("navigation", { name: "课程章节" })).toHaveTextContent("沿 Commit Graph 移动 HEAD");
  });

  it("runs a keyboard terminal command and judges the resulting state", async () => {
    const user = userEvent.setup();
    render(<GitCourseApp />);
    const terminal = screen.getByLabelText("Terminal output");
    const input = screen.getByLabelText("$");
    await user.type(input, "git add app.ts{enter}");
    expect(terminal).toHaveTextContent("staged app.ts");
    await user.click(screen.getByRole("button", { name: "Judge my state" }));
    expect(screen.getByRole("status")).toHaveTextContent("目标达成");
    expect(screen.getByRole("group")).not.toBeDisabled();
  });
});
