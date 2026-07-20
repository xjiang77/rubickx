export class PatternError extends Error { constructor(code) { super(code); this.code = code; } }

export function evaluate(input) {
    if (input.max_attempts < 1) throw new PatternError("invalid_attempt_budget");
    const processed = [], dead = [], receipts = [], byId = new Map();
    for (const message of input.messages ?? []) {
        if (byId.has(message.id)) throw new PatternError("duplicate_message_id");
        byId.set(message.id, message);
        if (!message.outcomes?.length) throw new PatternError("missing_outcome");
        for (let index=0; index<input.max_attempts; index++) {
            const outcome = message.outcomes[Math.min(index, message.outcomes.length-1)]; const attempt=index+1;
            if (outcome === "success") { processed.push(message.id); receipts.push(`processed:${message.id}:${attempt}`); break; }
            if (outcome === "permanent") { dead.push({id:message.id,reason:"permanent",attempts:attempt,status:"dead"}); receipts.push(`dead:${message.id}:permanent:${attempt}`); break; }
            if (outcome !== "transient") throw new PatternError("unknown_processing_outcome");
            if (attempt === input.max_attempts) { dead.push({id:message.id,reason:"transient_exhausted",attempts:attempt,status:"dead"}); receipts.push(`dead:${message.id}:transient_exhausted:${attempt}`); }
            else receipts.push(`retry:${message.id}:${attempt}`);
        }
    }
    for (const id of input.replay_ids ?? []) {
        const record = dead.find((item) => item.id === id && item.status === "dead");
        if (!record) throw new PatternError("replay_not_dead");
        const outcome = byId.get(id).replay_outcome ?? "failure";
        if (outcome === "success") { record.status="replayed"; processed.push(id); receipts.push(`replayed:${id}`); }
        else if (outcome === "failure") receipts.push(`replay_failed:${id}`);
        else throw new PatternError("unknown_replay_outcome");
    }
    return {processed, dead_letters:dead, receipts};
}
