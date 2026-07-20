class PatternError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code

def evaluate(input_data):
    items=input_data.get("items",[]);stages=input_data.get("stages",[]);outputs=[];receipts=[];limit=input_data.get("consumer_limit")
    for index,item in enumerate(items):
        if limit is not None and len(outputs)>=limit:
            return {"outputs":outputs,"status":"cancelled","failed_stage":"none","cancelled_items":items[index:],"stage_receipts":receipts,"upstream_cancelled":True}
        value=item
        for stage in stages:
            before=value
            if stage=="reject_negative":
                if value<0:return {"outputs":outputs,"status":"failed","failed_stage":stage,"cancelled_items":items[index+1:],"stage_receipts":receipts+[f"failed:{stage}:{value}"],"upstream_cancelled":True}
            elif stage=="double":value*=2
            elif stage=="increment":value+=1
            else:raise PatternError("unknown_stage")
            receipts.append(f"{stage}:{before}->{value}")
        outputs.append(value)
    return {"outputs":outputs,"status":"completed","failed_stage":"none","cancelled_items":[],"stage_receipts":receipts,"upstream_cancelled":False}
