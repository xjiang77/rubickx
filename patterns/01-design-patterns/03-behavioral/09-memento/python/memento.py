from dataclasses import dataclass
class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
@dataclass(frozen=True)
class Memento:primary:str;fallbacks:tuple[str,...]
class RouteConfig:
    def __init__(self,value):self.primary=value["primary"];self.fallbacks=list(value.get("fallbacks",[]))
    def snapshot(self):return Memento(self.primary,tuple(self.fallbacks))
    def restore(self,memento):self.primary=memento.primary;self.fallbacks=list(memento.fallbacks)
    def value(self):return{"primary":self.primary,"fallbacks":list(self.fallbacks)}
def evaluate(input_data):
    originator=RouteConfig(input_data["initial"]);snapshots=[];audit=[]
    for operation in input_data.get("operations",[]):
        name=operation["op"]
        if name=="snapshot":snapshots.append(originator.snapshot());audit.append(f"snapshot:{len(snapshots)-1}")
        elif name=="set_primary":originator.primary=operation["value"];audit.append(f"set_primary:{operation['value']}")
        elif name=="append_fallback":originator.fallbacks.append(operation["value"]);audit.append(f"append_fallback:{operation['value']}")
        elif name=="restore":
            index=operation["index"]
            if index<0 or index>=len(snapshots):raise PatternError("unknown_memento")
            originator.restore(snapshots[index]);audit.append(f"restore:{index}")
        else:raise PatternError("unsupported_operation")
    return{"config":originator.value(),"snapshots":len(snapshots),"audit":audit}
