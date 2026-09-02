class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class Handler:
    name=""
    def __init__(self,next_handler=None):self.next=next_handler
    def forward(self,request,visited):return self.next.handle(request,visited) if self.next else None
class Auth(Handler):
    name="auth"
    def handle(self,request,visited):
        visited.append(self.name)
        if request.get("token")!="valid":raise PatternError("unauthenticated")
        return self.forward(request,visited)
class Quota(Handler):
    name="quota"
    def handle(self,request,visited):
        visited.append(self.name)
        if request.get("quota",0)<=0:raise PatternError("quota_exhausted")
        return self.forward(request,visited)
class Execute(Handler):
    name="execute"
    def handle(self,request,visited):visited.append(self.name);return f"accepted:{request['payload']}"
def evaluate(input_data):
    chain=Auth(Quota(Execute()));responses=[]
    for request in input_data.get("requests",[]):
        visited=[];responses.append({"decision":chain.handle(request,visited),"visited":visited})
    return{"responses":responses}
