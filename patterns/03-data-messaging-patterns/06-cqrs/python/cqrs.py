class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def evaluate(input_data):
    events, seen, write_balance = [], set(), 0
    for command in input_data.get("commands", []):
        if command["id"] in seen:
            raise PatternError("duplicate_command_id")
        seen.add(command["id"]); write_balance += command["delta"]
        events.append({"version": len(events)+1, "delta": command["delta"]})
    projection_balance = projection_version = 0
    snapshots = []
    for target in input_data.get("projection_targets", []):
        if target < projection_version:
            raise PatternError("projection_regression")
        if target > len(events):
            raise PatternError("projection_ahead")
        for event in events[projection_version:target]:
            projection_balance += event["delta"]
        projection_version = target
        snapshots.append({"balance": projection_balance, "version": projection_version, "lag": len(events)-projection_version})
    return {"write_model": {"balance": write_balance, "version": len(events)}, "projection_snapshots": snapshots}
