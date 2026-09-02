class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def evaluate(input_data):
    inbox = {}
    receipts = []
    balance = 0
    for message in input_data.get("messages", []):
        message_id = message["id"]
        amount = message["amount"]
        if message_id in inbox:
            if inbox[message_id] != amount:
                raise PatternError("message_identity_conflict")
            receipts.append(f"duplicate:{message_id}")
        else:
            inbox[message_id] = amount
            balance += amount
            receipts.append(f"applied:{message_id}")
    return {"receipts": receipts, "balance": balance, "inbox_ids": sorted(inbox)}
