class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class IngestionJob:
    format=""
    def run(self,payload):
        if not payload:raise PatternError("invalid_payload")
        steps=["validate"];data=self.transform(payload);steps.append(f"transform:{self.format}");steps.append("persist");return{"format":self.format,"data":data,"steps":steps}
    def transform(self,payload):raise NotImplementedError
class CsvJob(IngestionJob):
    format="csv"
    def transform(self,payload):return payload.split(",")
class JsonJob(IngestionJob):
    format="json"
    def transform(self,payload):return payload.split("|")
def evaluate(input_data):
    results=[]
    for value in input_data.get("jobs",[]):
        job=CsvJob() if value["format"]=="csv" else JsonJob() if value["format"]=="json" else (_ for _ in ()).throw(PatternError("unsupported_format"));results.append(job.run(value.get("payload","")))
    return{"results":results}
