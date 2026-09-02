class PatternError(Exception):
    def __init__(self, code): super().__init__(code); self.code = code

class ArtifactValidator:
    def validate(self, artifact):
        if not artifact: raise PatternError("invalid_artifact")
        return f"validate:{artifact}"

class Deployer:
    def deploy(self, artifact): return f"deploy:{artifact}"

class HealthVerifier:
    def verify(self, artifact, healthy):
        if not healthy: raise PatternError("health_check_failed")
        return f"verify:{artifact}"

class ReleaseFacade:
    def __init__(self):
        self.validator, self.deployer, self.verifier = ArtifactValidator(), Deployer(), HealthVerifier()
    def release(self, request):
        artifact = request.get("artifact", "")
        steps = [self.validator.validate(artifact)]
        if request.get("dry_run", False):
            steps.append(f"plan:{artifact}")
            return {"artifact": artifact, "status": "planned", "steps": steps}
        steps.append(self.deployer.deploy(artifact))
        steps.append(self.verifier.verify(artifact, request.get("healthy", False)))
        return {"artifact": artifact, "status": "released", "steps": steps}

def evaluate(input_data):
    facade = ReleaseFacade()
    return {"releases": [facade.release(item) for item in input_data.get("releases", [])]}
