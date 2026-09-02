class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def evaluate(input_data):
    count = input_data["worker_count"]
    if count < 1:
        raise PatternError("invalid_worker_count")
    available = [0] * count
    used, executions = set(), []
    for job in input_data.get("jobs", []):
        if job["duration"] < 1:
            raise PatternError("invalid_duration")
        if job["outcome"] not in ("success", "failure"):
            raise PatternError("unknown_job_outcome")
        worker = min(range(count), key=lambda index: (available[index], index))
        start = available[worker]; finish = start + job["duration"]; available[worker] = finish; used.add(worker)
        executions.append({"id": job["id"], "worker": worker, "start": start, "finish": finish, "status": "completed" if job["outcome"] == "success" else "failed"})
    return {"executions": executions, "workers_used": len(used), "makespan": max(available, default=0), "joined": True}
