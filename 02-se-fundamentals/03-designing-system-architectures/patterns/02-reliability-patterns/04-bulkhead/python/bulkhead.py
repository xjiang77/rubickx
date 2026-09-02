class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
def evaluate(input_data):
    capacities=dict(input_data.get("capacities",{}));used={key:0 for key in capacities};decisions=[]
    for action in input_data.get("actions",[]):
        pool=action["pool"]
        if pool not in capacities:raise PatternError("unknown_pool")
        if action["op"]=="acquire":
            if used[pool]>=capacities[pool]:decisions.append(f"rejected:{pool}")
            else:used[pool]+=1;decisions.append(f"accepted:{pool}")
        elif action["op"]=="release":
            if used[pool]<=0:raise PatternError("over_release")
            used[pool]-=1;decisions.append(f"released:{pool}")
        else:raise PatternError("unsupported_action")
    return{"decisions":decisions,"used":dict(sorted(used.items()))}
