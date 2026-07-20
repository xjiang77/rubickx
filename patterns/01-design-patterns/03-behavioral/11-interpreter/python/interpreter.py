class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
def resolve(token,context):
    try:return int(token)
    except ValueError:
        if token not in context:raise PatternError("unknown_identifier")
        return context[token]
class Comparison:
    def __init__(self,left,operator,right):self.left=left;self.operator=operator;self.right=right
    def interpret(self,context,counter):
        counter[0]+=1;left,right=resolve(self.left,context),resolve(self.right,context)
        return left>=right if self.operator==">=" else left==right
class And:
    def __init__(self,children):self.children=children
    def interpret(self,context,counter):return all(child.interpret(context,counter) for child in self.children)
def parse(expression):
    tokens=expression.split();children=[];index=0
    while index<len(tokens):
        if index+2>=len(tokens):raise PatternError("invalid_expression")
        left,operator,right=tokens[index:index+3]
        if operator not in {">=","=="}:raise PatternError("unsupported_operator")
        children.append(Comparison(left,operator,right));index+=3
        if index<len(tokens):
            if tokens[index]!="and":raise PatternError("invalid_expression")
            index+=1
    if not children:raise PatternError("invalid_expression")
    return And(children)
def evaluate(input_data):
    counter=[0];allowed=parse(input_data["expression"]).interpret(input_data.get("context",{}),counter);return{"allowed":allowed,"comparisons":counter[0]}
