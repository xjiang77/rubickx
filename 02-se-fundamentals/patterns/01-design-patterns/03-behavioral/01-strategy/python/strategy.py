class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class MetricStrategy:
    field=""
    def select(self,candidates):
        healthy=[value for value in candidates if value.get("healthy",False)]
        if not healthy:raise PatternError("no_healthy_candidate")
        return min(healthy,key=lambda value:(value[self.field],value["name"]))["name"]
class CostStrategy(MetricStrategy):field="cost"
class LatencyStrategy(MetricStrategy):field="latency"
def evaluate(input_data):
    strategies={"cost":CostStrategy(),"latency":LatencyStrategy()};selected=[]
    for value in input_data.get("selections",[]):
        strategy=strategies.get(value.get("strategy"))
        if strategy is None:raise PatternError("unsupported_strategy")
        selected.append(strategy.select(value.get("candidates",[])))
    return{"selected":selected}
