class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class SupportMediator:
    routes={"customer":["agent","bot"],"agent":["customer"],"bot":["agent"]}
    def dispatch(self,event):
        recipients=self.routes.get(event.get("from"))
        if recipients is None:raise PatternError("unsupported_sender")
        return[{"from":event["from"],"to":target,"message":event["message"]} for target in recipients]
def evaluate(input_data):
    mediator=SupportMediator();deliveries=[]
    for event in input_data.get("events",[]):deliveries.extend(mediator.dispatch(event))
    return{"deliveries":deliveries}
