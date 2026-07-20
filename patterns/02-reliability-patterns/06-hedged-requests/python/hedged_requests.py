class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
def evaluate(input_data):
    if not input_data.get("idempotent",False):raise PatternError("hedge_not_allowed")
    primary=input_data["primary"]
    if primary["result"]=="success" and primary["latency_ms"]<=input_data["hedge_delay_ms"]:return{"winner":"primary","completed_ms":primary["latency_ms"],"attempts":1,"cancelled":[]}
    start=min(input_data["hedge_delay_ms"],primary["latency_ms"] if primary["result"]=="failure" else input_data["hedge_delay_ms"]);hedge=input_data["hedge"];candidates=[]
    if primary["result"]=="success":candidates.append((primary["latency_ms"],"primary"))
    if hedge["result"]=="success":candidates.append((start+hedge["latency_ms"],"hedge"))
    if not candidates:raise PatternError("all_attempts_failed")
    completed,winner=min(candidates);cancelled=[name for _,name in candidates if name!=winner and completed<dict((name,time) for time,name in candidates)[name]];return{"winner":winner,"completed_ms":completed,"attempts":2,"cancelled":cancelled}
