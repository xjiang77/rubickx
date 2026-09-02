class FakeClock:
    def __init__(self):self.now=0
    def advance(self,value):self.now+=value
def evaluate(input_data):
    clock=FakeClock();deadline=input_data["deadline_ms"];outcomes=[]
    for operation in input_data.get("operations",[]):
        finish=clock.now+operation["duration_ms"]
        if finish<=deadline:clock.advance(operation["duration_ms"]);outcomes.append({"name":operation["name"],"status":"completed","outcome":"success"})
        else:clock.advance(max(0,deadline-clock.now));outcomes.append({"name":operation["name"],"status":"timed_out","outcome":"unknown" if operation.get("side_effecting",False) else "abandoned"});break
    return{"outcomes":outcomes,"now_ms":clock.now}
