class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class Equals:
    def __init__(self,field,value):self.field=field;self.value=value
    def accept(self,visitor):return visitor.visit_equals(self)
class And:
    def __init__(self,children):self.children=children
    def accept(self,visitor):return visitor.visit_and(self)
class EvaluateVisitor:
    def __init__(self,context):self.context=context
    def visit_equals(self,node):return self.context.get(node.field)==node.value
    def visit_and(self,node):return all(child.accept(self) for child in node.children)
class FieldVisitor:
    def visit_equals(self,node):return[node.field]
    def visit_and(self,node):return[field for child in node.children for field in child.accept(self)]
def build(value):
    if value["type"]=="equals":return Equals(value["field"],value.get("value"))
    if value["type"]=="and":return And([build(child) for child in value.get("children",[])])
    raise PatternError("unsupported_node")
def evaluate(input_data):
    root=build(input_data["tree"])
    if input_data["operation"]=="evaluate":return{"result":root.accept(EvaluateVisitor(input_data.get("context",{})))}
    if input_data["operation"]=="collect_fields":return{"fields":root.accept(FieldVisitor())}
    raise PatternError("unsupported_operation")
