export class PatternError extends Error { constructor(code) { super(code); this.code = code; } }

export function evaluate(input) {
    const attempts = input.relay_attempts ?? [];
    if (input.commit !== true) {
        if (attempts.length > 0) throw new PatternError("relay_without_commit");
        return {aggregate_state: "absent", outbox_status: "absent", deliveries: [], relay_receipts: []};
    }
    const messageId = input.message_id;
    let status = "pending";
    const deliveries = [];
    const receipts = [];
    for (const attempt of attempts) {
        if (status === "sent") { receipts.push(`skipped_already_sent:${messageId}`); continue; }
        if (attempt === "crash_before_publish") receipts.push(`crash_before_publish:${messageId}`);
        else if (attempt === "crash_after_publish") { deliveries.push(messageId); receipts.push(`crash_after_publish:${messageId}`); }
        else if (attempt === "success") { deliveries.push(messageId); receipts.push(`published:${messageId}`); status = "sent"; }
        else throw new PatternError("unknown_relay_outcome");
    }
    return {aggregate_state: input.new_state, outbox_status: status, deliveries, relay_receipts: receipts};
}
