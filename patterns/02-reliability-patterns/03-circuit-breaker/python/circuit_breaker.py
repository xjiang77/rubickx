class Breaker:
    def __init__(self,threshold,cooldown):self.threshold=threshold;self.cooldown=cooldown;self.state="closed";self.failures=0;self.now=0;self.opened_at=0;self.forwarded=0
    def advance(self,value):self.now+=value
    def call(self,result):
        if self.state=="open":
            if self.now-self.opened_at<self.cooldown:return"rejected"
            self.state="half_open"
        self.forwarded+=1
        if self.state=="half_open":
            if result=="success":self.state="closed";self.failures=0;return"probe_success"
            self.state="open";self.opened_at=self.now;return"probe_failed"
        if result=="success":self.failures=0;return"success"
        self.failures+=1
        if self.failures>=self.threshold:self.state="open";self.opened_at=self.now;return"opened"
        return"failure"
def evaluate(input_data):
    breaker=Breaker(input_data["threshold"],input_data["cooldown"]);decisions=[]
    for event in input_data.get("events",[]):
        if "advance" in event:breaker.advance(event["advance"]);decisions.append("advanced")
        else:decisions.append(breaker.call(event["result"]))
    return{"decisions":decisions,"final":breaker.state,"forwarded":breaker.forwarded}
