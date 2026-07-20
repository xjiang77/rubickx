class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
def evaluate(input_data):
    children=input_data.get("children",[]);names=[child["name"] for child in children];order=input_data.get("completion_order",[])
    if len(set(names))!=len(names):raise PatternError("duplicate_child")
    if len(order)!=len(names) or len(set(order))!=len(order) or set(order)!=set(names):raise PatternError("invalid_completion_order")
    by_name={child["name"]:child for child in children};states={name:"pending" for name in names};results=[];status="completed";completed=0;cancel_after=input_data.get("parent_cancel_after")
    for name in order:
        if cancel_after is not None and completed>=cancel_after:
            status="cancelled";break
        child=by_name[name]
        if child["outcome"]=="success":states[name]="completed";results.append(f'{name}:{child["value"]}');completed+=1
        elif child["outcome"]=="failure":states[name]="failed";status="failed";break
        else:raise PatternError("unknown_outcome")
    if status!="completed":
        for name in names:
            if states[name]=="pending":states[name]="cancelled"
    return {"parent_status":status,"child_states":[f"{name}:{states[name]}" for name in names],"results":results,"joined_count":len(names),"leaked":0}
