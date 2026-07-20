class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def apply_event(balance, event):
    amount = event.get("amount", 0)
    if amount <= 0:
        raise PatternError("invalid_event_amount")
    if event["type"] == "deposited":
        return balance + amount
    if event["type"] == "withdrawn":
        if amount > balance:
            raise PatternError("invalid_event_history")
        return balance - amount
    raise PatternError("unknown_event_type")

def evaluate(input_data):
    events = input_data.get("events", [])
    balance = 0
    for event in events:
        balance = apply_event(balance, event)
    appended = []
    command = input_data.get("command")
    if command is not None:
        if command["expected_version"] != len(events):
            raise PatternError("version_conflict")
        amount = command["amount"]
        if amount <= 0:
            raise PatternError("invalid_command_amount")
        if command["type"] == "deposit":
            event_type = "deposited"
        elif command["type"] == "withdraw":
            if amount > balance:
                raise PatternError("insufficient_funds")
            event_type = "withdrawn"
        else:
            raise PatternError("unknown_command_type")
        new_event = {"type": event_type, "amount": amount, "version": len(events)+1}
        balance = apply_event(balance, new_event); appended.append(new_event)
    return {"balance": balance, "version": len(events)+len(appended), "history_count": len(events)+len(appended), "appended_events": appended}
