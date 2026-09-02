class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def evaluate(input_data):
    attempts = input_data.get("relay_attempts", [])
    if not input_data.get("commit", False):
        if attempts:
            raise PatternError("relay_without_commit")
        return {"aggregate_state": "absent", "outbox_status": "absent", "deliveries": [], "relay_receipts": []}
    state = input_data["new_state"]
    message_id = input_data["message_id"]
    status = "pending"
    deliveries = []
    receipts = []
    for attempt in attempts:
        if status == "sent":
            receipts.append(f"skipped_already_sent:{message_id}")
        elif attempt == "crash_before_publish":
            receipts.append(f"crash_before_publish:{message_id}")
        elif attempt == "crash_after_publish":
            deliveries.append(message_id)
            receipts.append(f"crash_after_publish:{message_id}")
        elif attempt == "success":
            deliveries.append(message_id)
            receipts.append(f"published:{message_id}")
            status = "sent"
        else:
            raise PatternError("unknown_relay_outcome")
    return {"aggregate_state": state, "outbox_status": status, "deliveries": deliveries, "relay_receipts": receipts}
