from dataclasses import dataclass
class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
@dataclass(frozen=True)
class ModelMetadata:model:str;provider:str;context_window:int
class FlyweightFactory:
    def __init__(self):self.values={}
    def register(self,value):
        candidate=ModelMetadata(value["model"],value["provider"],value["context_window"]);current=self.values.get(candidate.model)
        if current is not None and current!=candidate:raise PatternError("intrinsic_conflict")
        self.values[candidate.model]=candidate
    def get(self,model):
        if model not in self.values:raise PatternError("unknown_model")
        return self.values[model]
def evaluate(input_data):
    factory=FlyweightFactory()
    for value in input_data.get("definitions",[]):factory.register(value)
    flyweights=[];routes=[]
    for route in input_data.get("routes",[]):
        metadata=factory.get(route["model"]);flyweights.append(metadata);routes.append({"model":metadata.model,"provider":metadata.provider,"context_window":metadata.context_window,"tenant":route["tenant"]})
    reused=len(flyweights)>1 and all(value is flyweights[0] for value in flyweights)
    return{"routes":routes,"flyweight_count":len(factory.values),"reused":reused}
