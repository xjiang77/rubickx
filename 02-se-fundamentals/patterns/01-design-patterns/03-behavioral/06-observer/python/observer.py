class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class Observer:
    def __init__(self,name):self.name=name
    def notify(self,event):
        if self.name=="failing":raise PatternError("observer_failed")
class Subject:
    def __init__(self,observers):self.observers=list(observers)
    def publish(self,event):
        receipts=[]
        for observer in list(self.observers):
            try:observer.notify(event);receipts.append({"event":event,"observer":observer.name,"status":"delivered"})
            except PatternError as error:receipts.append({"event":event,"observer":observer.name,"status":"failed","error":error.code})
        return receipts
    def unsubscribe(self,names):self.observers=[value for value in self.observers if value.name not in names]
def evaluate(input_data):
    supported={"audit","metrics","failing"}
    if any(name not in supported for name in input_data.get("observers",[])):raise PatternError("unsupported_observer")
    subject=Subject([Observer(name) for name in input_data.get("observers",[])]);receipts=[]
    for index,event in enumerate(input_data.get("events",[])):
        receipts.extend(subject.publish(event))
        if index==0:subject.unsubscribe(set(input_data.get("unsubscribe_after_first",[])))
    return{"receipts":receipts,"active":len(subject.observers)}
