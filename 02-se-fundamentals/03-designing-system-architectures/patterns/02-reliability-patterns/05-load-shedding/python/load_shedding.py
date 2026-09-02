class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
def evaluate(input_data):
    rank={"high":0,"normal":1,"low":2};requests=input_data.get("requests",[])
    if any(value.get("priority") not in rank for value in requests):raise PatternError("unknown_priority")
    ordered=sorted(enumerate(requests),key=lambda pair:(rank[pair[1]["priority"]],pair[0]));capacity=max(0,input_data["capacity"]);accepted=[value["id"] for _,value in ordered[:capacity]];accepted_set=set(accepted);shed=[value["id"] for value in requests if value["id"] not in accepted_set];return{"accepted":accepted,"shed":shed,"goodput":len(accepted)}
