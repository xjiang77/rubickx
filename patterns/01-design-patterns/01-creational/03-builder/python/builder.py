from dataclasses import asdict, dataclass

class PatternError(Exception):
    def __init__(self, code): super().__init__(code); self.code = code

@dataclass(frozen=True)
class ChatRequest:
    model: str; messages: tuple[str, ...]; max_tokens: int; stream: bool

class ChatRequestBuilder:
    def __init__(self): self.reset()
    def reset(self): self.model=""; self.messages=[]; self.max_tokens=0; self.stream=False; return self
    def configure(self, value):
        self.model=value.get("model", ""); self.messages=list(value.get("messages", [])); self.max_tokens=value.get("max_tokens", 0); self.stream=value.get("stream", False); return self
    def build(self):
        if not self.model or not self.messages or self.max_tokens <= 0: raise PatternError("invalid_request")
        product=ChatRequest(self.model, tuple(self.messages), self.max_tokens, self.stream); self.reset(); return product

def evaluate(input_data):
    builder=ChatRequestBuilder(); requests=[]
    for value in input_data.get("builds", []):
        product=builder.configure(value).build(); item=asdict(product); item["messages"]=list(product.messages); requests.append(item)
    return {"requests": requests}
