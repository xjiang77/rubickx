class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class RouteTable:
    def __init__(self,initial):self.routes=dict(initial)
class SetRoute:
    def __init__(self,receiver,key,value):self.receiver=receiver;self.key=key;self.value=value;self.existed=False;self.previous=None
    def execute(self):self.existed=self.key in self.receiver.routes;self.previous=self.receiver.routes.get(self.key);self.receiver.routes[self.key]=self.value
    def undo(self):
        if self.existed:self.receiver.routes[self.key]=self.previous
        else:self.receiver.routes.pop(self.key,None)
class Invoker:
    def __init__(self):self.history=[]
    def run(self,command):command.execute();self.history.append(command)
    def undo(self):
        if not self.history:raise PatternError("no_history")
        self.history.pop().undo()
def evaluate(input_data):
    table=RouteTable(input_data.get("initial",{}));invoker=Invoker()
    for value in input_data.get("commands",[]):invoker.run(SetRoute(table,value["key"],value["value"]))
    for _ in range(input_data.get("undo",0)):invoker.undo()
    return{"routes":dict(sorted(table.routes.items())),"history":len(invoker.history)}
