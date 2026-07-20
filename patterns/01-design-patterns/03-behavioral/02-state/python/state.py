class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class State:
    name="";transitions={}
    def handle(self,event):
        target=self.transitions.get(event)
        if target is None:raise PatternError("invalid_transition")
        return STATES[target]
class Queued(State):name="queued";transitions={"start":"running","cancel":"cancelled"}
class Running(State):name="running";transitions={"complete":"completed","fail":"failed"}
class Failed(State):name="failed";transitions={"retry":"running","cancel":"cancelled"}
class Completed(State):name="completed"
class Cancelled(State):name="cancelled"
STATES={value.name:value() for value in (Queued,Running,Failed,Completed,Cancelled)}
def evaluate(input_data):
    state=STATES.get(input_data.get("initial"))
    if state is None:raise PatternError("unknown_state")
    history=[state.name]
    for event in input_data.get("events",[]):state=state.handle(event);history.append(state.name)
    return{"final":state.name,"history":history}
