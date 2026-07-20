class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def evaluate(input_data):
    subscribers = input_data.get("subscribers", [])
    names = [subscriber["name"] for subscriber in subscribers]
    if len(names) != len(set(names)):
        raise PatternError("duplicate_subscriber")
    deliveries = []
    events = input_data.get("events", [])
    for event in events:
        for subscriber in subscribers:
            if event["topic"] not in subscriber.get("topics", []):
                continue
            outcome = subscriber.get("outcome", "success")
            if outcome not in ("success", "failure"):
                raise PatternError("unknown_delivery_outcome")
            deliveries.append({"event_id": event["id"], "subscriber": subscriber["name"], "status": "delivered" if outcome == "success" else "failed"})
    return {"published": len(events), "deliveries": deliveries}
