export class PatternError extends Error { constructor(code){super(code);this.code=code;} }
class Validator { validate(artifact){if(!artifact)throw new PatternError("invalid_artifact");return `validate:${artifact}`;} }
class Deployer { deploy(artifact){return `deploy:${artifact}`;} }
class Verifier { verify(artifact,healthy){if(!healthy)throw new PatternError("health_check_failed");return `verify:${artifact}`;} }
class ReleaseFacade {
  constructor(){this.validator=new Validator();this.deployer=new Deployer();this.verifier=new Verifier();}
  release(request){const artifact=request.artifact??"";const steps=[this.validator.validate(artifact)];if(request.dry_run===true){steps.push(`plan:${artifact}`);return{artifact,status:"planned",steps};}steps.push(this.deployer.deploy(artifact));steps.push(this.verifier.verify(artifact,request.healthy===true));return{artifact,status:"released",steps};}
}
export function evaluate(input){const facade=new ReleaseFacade();return{releases:(input.releases??[]).map((item)=>facade.release(item))};}
