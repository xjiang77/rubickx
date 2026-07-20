class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def evaluate(input_data):
    capacity = input_data["capacity"]
    if capacity < 0:
        raise PatternError("invalid_capacity")
    queue, consumed, receipts, closed = [], [], [], False
    for action in input_data.get("actions", []):
        operation = action["op"]
        if operation == "produce":
            item = action["item"]
            if closed:
                raise PatternError("produce_after_close")
            if len(queue) >= capacity:
                receipts.append(f"backpressured:{item}")
            else:
                queue.append(item); receipts.append(f"queued:{item}")
        elif operation == "consume":
            if queue:
                item = queue.pop(0); consumed.append(item); receipts.append(f"consumed:{item}")
            else:
                receipts.append("end" if closed else "empty")
        elif operation == "close":
            if closed:
                raise PatternError("already_closed")
            closed = True; receipts.append("closed")
        else:
            raise PatternError("unknown_action")
    return {"receipts": receipts, "consumed": consumed, "remaining": queue, "closed": closed}
