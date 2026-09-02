class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def evaluate(input_data):
    budget = input_data["max_attempts"]
    if budget < 1:
        raise PatternError("invalid_attempt_budget")
    processed, dead, receipts = [], [], []
    by_id = {}
    for message in input_data.get("messages", []):
        if message["id"] in by_id:
            raise PatternError("duplicate_message_id")
        by_id[message["id"]] = message
        outcomes = message.get("outcomes", [])
        if not outcomes:
            raise PatternError("missing_outcome")
        for index in range(budget):
            outcome = outcomes[index] if index < len(outcomes) else outcomes[-1]
            attempt = index + 1
            if outcome == "success":
                processed.append(message["id"]); receipts.append(f"processed:{message['id']}:{attempt}"); break
            if outcome == "permanent":
                dead.append({"id": message["id"], "reason": "permanent", "attempts": attempt, "status": "dead"}); receipts.append(f"dead:{message['id']}:permanent:{attempt}"); break
            if outcome != "transient":
                raise PatternError("unknown_processing_outcome")
            if attempt == budget:
                dead.append({"id": message["id"], "reason": "transient_exhausted", "attempts": attempt, "status": "dead"}); receipts.append(f"dead:{message['id']}:transient_exhausted:{attempt}")
            else:
                receipts.append(f"retry:{message['id']}:{attempt}")
    for message_id in input_data.get("replay_ids", []):
        record = next((item for item in dead if item["id"] == message_id and item["status"] == "dead"), None)
        if record is None:
            raise PatternError("replay_not_dead")
        outcome = by_id[message_id].get("replay_outcome", "failure")
        if outcome == "success":
            record["status"] = "replayed"; processed.append(message_id); receipts.append(f"replayed:{message_id}")
        elif outcome == "failure":
            receipts.append(f"replay_failed:{message_id}")
        else:
            raise PatternError("unknown_replay_outcome")
    return {"processed": processed, "dead_letters": dead, "receipts": receipts}
