class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class BaseClient:
    def send(self,request):return{"path":request["path"],"headers":dict(request.get("headers",{})),"applied":list(request.get("applied",[]))}
class AuthDecorator:
    def __init__(self,next_client,token):self.next=next_client;self.token=token
    def send(self,request):
        if not self.token:raise PatternError("missing_token")
        value={**request,"headers":dict(request.get("headers",{})),"applied":list(request.get("applied",[]))};value["headers"]["Authorization"]=f"Bearer {self.token}";value["applied"].append("auth");return self.next.send(value)
class TraceDecorator:
    def __init__(self,next_client):self.next=next_client
    def send(self,request):
        trace=request.get("trace_id")
        if not trace:raise PatternError("missing_trace")
        value={**request,"headers":dict(request.get("headers",{})),"applied":list(request.get("applied",[]))};value["headers"]["X-Trace-Id"]=trace;value["applied"].append("trace");return self.next.send(value)
def evaluate(input_data):
    client=BaseClient()
    for name in reversed(input_data.get("decorators",[])):
        client=AuthDecorator(client,input_data.get("token")) if name=="auth" else TraceDecorator(client) if name=="trace" else (_ for _ in ()).throw(PatternError("unsupported_decorator"))
    return{"responses":[client.send(request) for request in input_data.get("requests",[])]}
