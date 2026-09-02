class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class DocumentStore:
    def __init__(self):self.documents={"public":"status:ok","secret":"key:rotated"}
    def read(self,name):
        if name not in self.documents:raise PatternError("not_found")
        return self.documents[name]
class DocumentProxy:
    def __init__(self,role):self.role=role;self.subject=None;self.load_count=0
    def read(self,name):
        if name=="secret" and self.role!="admin":raise PatternError("forbidden")
        if self.subject is None:self.subject=DocumentStore();self.load_count+=1
        return self.subject.read(name)
def evaluate(input_data):
    proxy=DocumentProxy(input_data.get("role","viewer"));values=[proxy.read(name) for name in input_data.get("reads",[])];return{"values":values,"load_count":proxy.load_count}
