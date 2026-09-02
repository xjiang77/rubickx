class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def evaluate(input_data):
    completed = []
    actions = []
    for step in input_data.get("steps", []):
        name = step["name"]
        actions.append(f"execute:{name}")
        result = step["result"]
        if result == "success":
            completed.append(step)
            continue
        if result != "failure":
            raise PatternError("unknown_step_result")
        actions.append(f"failed:{name}")
        failed_compensations = []
        for prior in reversed(completed):
            outcome = prior.get("compensation", "success")
            if outcome not in ("success", "failure"):
                raise PatternError("unknown_compensation_result")
            actions.append(f"compensate:{prior['name']}:{outcome}")
            if outcome == "failure":
                failed_compensations.append(prior["name"])
        return {"status": "recovery_required" if failed_compensations else "compensated", "completed": [item["name"] for item in completed], "failed_step": name, "failed_compensations": failed_compensations, "actions": actions}
    return {"status": "completed", "completed": [item["name"] for item in completed], "failed_step": "none", "failed_compensations": [], "actions": actions}
