class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
def evaluate(input_data):
    state="pending";value="none";error="none";observations=[];terminal_count=0
    for action in input_data.get("actions",[]):
        op=action["op"]
        if op in ("complete","fail","cancel"):
            if state!="pending":raise PatternError("already_completed")
            terminal_count+=1
            if op=="complete":state="fulfilled";value=action["value"]
            elif op=="fail":state="rejected";error=action["code"]
            else:state="cancelled"
        elif op=="observe": observations.append("pending" if state=="pending" else f"value:{value}" if state=="fulfilled" else f"error:{error}" if state=="rejected" else "cancelled")
        else:raise PatternError("unknown_action")
    return {"state":state,"observations":observations,"value":value,"error":error,"terminal_count":terminal_count}
