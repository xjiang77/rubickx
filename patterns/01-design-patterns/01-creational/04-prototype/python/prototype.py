from dataclasses import dataclass
class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
@dataclass
class RoutingPolicy:
    name:str;primary:str;fallbacks:list[str]
    def clone(self):return RoutingPolicy(self.name+"-copy",self.primary,list(self.fallbacks))
    def value(self):return{"name":self.name,"primary":self.primary,"fallbacks":list(self.fallbacks)}
def evaluate(input_data):
    value=input_data["base"]
    if not value.get("name") or not value.get("primary"):raise PatternError("invalid_prototype")
    original=RoutingPolicy(value["name"],value["primary"],list(value.get("fallbacks",[])));clone=original.clone()
    if "override_primary" in input_data:clone.primary=input_data["override_primary"]
    if "append_fallback" in input_data:clone.fallbacks.append(input_data["append_fallback"])
    return{"original":original.value(),"clone":clone.value()}
