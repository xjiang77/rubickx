class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class EmailChannel:
    def send(self,recipient,message):return f"email:{recipient}:{message}"
class SlackChannel:
    def send(self,recipient,message):return f"slack:{recipient}:{message}"
class Alert:
    prefix=""
    def __init__(self,channel):self.channel=channel
    def deliver(self,recipient,body):return self.channel.send(recipient,f"{self.prefix} {body}")
class IncidentAlert(Alert):prefix="[INCIDENT]"
class ReminderAlert(Alert):prefix="[REMINDER]"
def evaluate(input_data):
    deliveries=[]
    for item in input_data.get("notifications",[]):
        channel=EmailChannel() if item["channel"]=="email" else SlackChannel() if item["channel"]=="slack" else (_ for _ in ()).throw(PatternError("unsupported_channel"))
        alert=IncidentAlert(channel) if item["kind"]=="incident" else ReminderAlert(channel) if item["kind"]=="reminder" else (_ for _ in ()).throw(PatternError("unsupported_alert"))
        deliveries.append(alert.deliver(item["recipient"],item["body"]))
    return{"deliveries":deliveries}
