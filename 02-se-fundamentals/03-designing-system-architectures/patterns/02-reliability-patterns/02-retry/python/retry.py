class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class FakeScheduler:
    def __init__(self):self.delays=[]
    def wait(self,value):self.delays.append(value)
def evaluate(input_data):
    scheduler=FakeScheduler();attempts=0;outcomes=input_data.get("outcomes",[])
    while attempts<input_data["max_attempts"]:
        outcome=outcomes[attempts] if attempts<len(outcomes) else "transient";attempts+=1
        if outcome=="success":return{"status":"success","attempts":attempts,"delays_ms":scheduler.delays}
        if outcome=="permanent":raise PatternError("non_retryable")
        if outcome!="transient":raise PatternError("unknown_outcome")
        if attempts<input_data["max_attempts"]:scheduler.wait(input_data["base_delay_ms"]*(2**(attempts-1)))
    return{"status":"exhausted","attempts":attempts,"delays_ms":scheduler.delays}
