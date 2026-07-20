class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class Equals:
    def __init__(self,field,value):self.field=field;self.value=value
    def evaluate(self,context,trace):trace.append(self.field);return context.get(self.field)==self.value
class All:
    def __init__(self,children):self.children=children
    def evaluate(self,context,trace):return all(child.evaluate(context,trace) for child in self.children)
class AnyOf:
    def __init__(self,children):self.children=children
    def evaluate(self,context,trace):return any(child.evaluate(context,trace) for child in self.children)
def build(value):
    kind=value.get("type")
    if kind=="equals":return Equals(value["field"],value.get("value"))
    if kind in ("all","any"):return (All if kind=="all" else AnyOf)([build(child) for child in value.get("children",[])])
    raise PatternError("unsupported_node")
def evaluate(input_data):
    trace=[];allowed=build(input_data["tree"]).evaluate(input_data.get("context",{}),trace);return{"allowed":allowed,"evaluated":trace}
