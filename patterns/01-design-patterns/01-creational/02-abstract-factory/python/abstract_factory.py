from abc import ABC, abstractmethod

class PatternError(Exception):
    def __init__(self, code): super().__init__(code); self.code = code

class CloudFactory(ABC):
    family = ""
    @abstractmethod
    def queue(self, prefix): ...
    @abstractmethod
    def object_store(self, prefix): ...
    def resources(self, prefix):
        return {"family": self.family, "queue": self.queue(prefix), "object_store": self.object_store(prefix)}

class AwsFactory(CloudFactory):
    family = "aws"
    def queue(self, prefix): return f"sqs:{prefix}"
    def object_store(self, prefix): return f"s3:{prefix}"

class GcpFactory(CloudFactory):
    family = "gcp"
    def queue(self, prefix): return f"pubsub:{prefix}"
    def object_store(self, prefix): return f"gcs:{prefix}"

def evaluate(input_data):
    factories = {"aws": AwsFactory, "gcp": GcpFactory}
    resources = []
    for provider in input_data.get("providers", []):
        factory_type = factories.get(provider)
        if factory_type is None: raise PatternError("unsupported_provider")
        resources.append(factory_type().resources(input_data["prefix"]))
    return {"resources": resources}
