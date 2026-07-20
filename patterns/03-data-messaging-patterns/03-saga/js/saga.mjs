export class PatternError extends Error { constructor(code) { super(code); this.code = code; } }

export function evaluate(input) {
    const completed = [];
    const actions = [];
    for (const step of input.steps ?? []) {
        actions.push(`execute:${step.name}`);
        if (step.result === "success") { completed.push(step); continue; }
        if (step.result !== "failure") throw new PatternError("unknown_step_result");
        actions.push(`failed:${step.name}`);
        const failed = [];
        for (const prior of [...completed].reverse()) {
            const outcome = prior.compensation ?? "success";
            if (outcome !== "success" && outcome !== "failure") throw new PatternError("unknown_compensation_result");
            actions.push(`compensate:${prior.name}:${outcome}`);
            if (outcome === "failure") failed.push(prior.name);
        }
        return {status: failed.length ? "recovery_required" : "compensated", completed: completed.map((item) => item.name), failed_step: step.name, failed_compensations: failed, actions};
    }
    return {status: "completed", completed: completed.map((item) => item.name), failed_step: "none", failed_compensations: [], actions};
}
