export class PatternError extends Error { constructor(code) { super(code); this.code = code; } }

export function evaluate(input) {
    if (input.capacity < 0) throw new PatternError("invalid_capacity");
    const queue=[], consumed=[], receipts=[]; let closed=false;
    for (const action of input.actions ?? []) {
        if (action.op === "produce") {
            if (closed) throw new PatternError("produce_after_close");
            if (queue.length >= input.capacity) receipts.push(`backpressured:${action.item}`); else { queue.push(action.item); receipts.push(`queued:${action.item}`); }
        } else if (action.op === "consume") {
            if (queue.length) { const item=queue.shift(); consumed.push(item); receipts.push(`consumed:${item}`); } else receipts.push(closed ? "end" : "empty");
        } else if (action.op === "close") {
            if (closed) throw new PatternError("already_closed"); closed=true; receipts.push("closed");
        } else throw new PatternError("unknown_action");
    }
    return {receipts,consumed,remaining:queue,closed};
}
