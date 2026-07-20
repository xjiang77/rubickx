export class PatternError extends Error { constructor(code) { super(code); this.code = code; } }

export function evaluate(input) {
    const inbox = new Map();
    const receipts = [];
    let balance = 0;
    for (const message of input.messages ?? []) {
        if (inbox.has(message.id)) {
            if (inbox.get(message.id) !== message.amount) throw new PatternError("message_identity_conflict");
            receipts.push(`duplicate:${message.id}`);
        } else {
            inbox.set(message.id, message.amount);
            balance += message.amount;
            receipts.push(`applied:${message.id}`);
        }
    }
    return {receipts, balance, inbox_ids: [...inbox.keys()].sort()};
}
