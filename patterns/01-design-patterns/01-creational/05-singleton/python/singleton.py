from threading import Lock
class PatternError(Exception):
    def __init__(self,code):super().__init__(code);self.code=code
class ProviderCatalog:
    _instance=None;_lock=Lock()
    def __new__(cls):
        with cls._lock:
            if cls._instance is None:cls._instance=super().__new__(cls);cls._instance._entries={}
            return cls._instance
    def register(self,name,endpoint):
        current=self._entries.get(name)
        if current is not None and current!=endpoint:raise PatternError("registration_conflict")
        self._entries[name]=endpoint
    def entries(self):return dict(sorted(self._entries.items()))
    @classmethod
    def _reset_for_test(cls):
        with cls._lock:cls._instance=None
def evaluate(input_data):
    ProviderCatalog._reset_for_test();first=ProviderCatalog();second=ProviderCatalog()
    for value in input_data.get("registrations",[]):first.register(value["name"],value["endpoint"])
    entries=second.entries();return{"same_instance":first is second,"entries":entries,"size":len(entries)}
