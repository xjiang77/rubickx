export class PatternError extends Error { constructor(code) { super(code); this.code = code; } }

function applyEvent(balance,event) {
    if (event.amount <= 0) throw new PatternError("invalid_event_amount");
    if (event.type === "deposited") return balance+event.amount;
    if (event.type === "withdrawn") { if (event.amount > balance) throw new PatternError("invalid_event_history"); return balance-event.amount; }
    throw new PatternError("unknown_event_type");
}

export function evaluate(input) {
    const events=input.events??[]; let balance=0; for (const event of events) balance=applyEvent(balance,event); const appended=[];
    if (input.command !== undefined) {
        const command=input.command; if (command.expected_version !== events.length) throw new PatternError("version_conflict"); if (command.amount <= 0) throw new PatternError("invalid_command_amount");
        let eventType; if (command.type === "deposit") eventType="deposited"; else if (command.type === "withdraw") { if (command.amount > balance) throw new PatternError("insufficient_funds"); eventType="withdrawn"; } else throw new PatternError("unknown_command_type");
        const event={type:eventType,amount:command.amount,version:events.length+1}; balance=applyEvent(balance,event); appended.push(event);
    }
    const count=events.length+appended.length; return {balance,version:count,history_count:count,appended_events:appended};
}
