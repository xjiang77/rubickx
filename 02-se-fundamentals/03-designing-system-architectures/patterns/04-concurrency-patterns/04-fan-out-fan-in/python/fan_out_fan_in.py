class PatternError(Exception):
    def __init__(self, code): super().__init__(code); self.code=code
def evaluate(input_data):
    mode=input_data.get("mode"); tasks=input_data.get("tasks",[])
    if mode not in ("all","first_success"): raise PatternError("unknown_mode")
    if mode=="first_success" and input_data.get("side_effecting",False): raise PatternError("fanout_not_allowed")
    if len({task["id"] for task in tasks})!=len(tasks): raise PatternError("duplicate_task")
    ordered=sorted(enumerate(tasks),key=lambda pair:(pair[1]["complete_at"],pair[0])); completion=[];results=[];failures=[]
    for position,(_,task) in enumerate(ordered):
        completion.append(task["id"])
        if task["outcome"]=="success":
            results.append(f'{task["id"]}:{task["value"]}')
            if mode=="first_success": return {"completion_order":completion,"results":results,"failures":failures,"cancelled":[later[1]["id"] for later in ordered[position+1:]],"status":"completed"}
        elif task["outcome"]=="failure": failures.append(task["id"])
        else: raise PatternError("unknown_outcome")
    if mode=="first_success": raise PatternError("all_tasks_failed")
    return {"completion_order":completion,"results":results,"failures":failures,"cancelled":[],"status":"partial" if failures else "completed"}
