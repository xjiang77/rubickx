export class PatternError extends Error { constructor(code) { super(code); this.code = code; } }

export function evaluate(input) {
    const subscribers = input.subscribers ?? [];
    const names = new Set();
    for (const subscriber of subscribers) {
        if (names.has(subscriber.name)) throw new PatternError("duplicate_subscriber");
        names.add(subscriber.name);
    }
    const events = input.events ?? [];
    const deliveries = [];
    for (const event of events) {
        for (const subscriber of subscribers) {
            if (!(subscriber.topics ?? []).includes(event.topic)) continue;
            const outcome = subscriber.outcome ?? "success";
            if (outcome !== "success" && outcome !== "failure") throw new PatternError("unknown_delivery_outcome");
            deliveries.push({event_id: event.id, subscriber: subscriber.name, status: outcome === "success" ? "delivered" : "failed"});
        }
    }
    return {published: events.length, deliveries};
}
