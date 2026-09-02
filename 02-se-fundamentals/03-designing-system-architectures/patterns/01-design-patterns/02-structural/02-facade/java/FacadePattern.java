import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class FacadePattern {
    static final class PatternException extends RuntimeException { private final String code; PatternException(String code){super(code);this.code=code;} String code(){return code;} }
    static final class Validator { String validate(String artifact){if(artifact.isEmpty())throw new PatternException("invalid_artifact");return "validate:"+artifact;} }
    static final class Deployer { String deploy(String artifact){return "deploy:"+artifact;} }
    static final class Verifier { String verify(String artifact, boolean healthy){if(!healthy)throw new PatternException("health_check_failed");return "verify:"+artifact;} }
    static final class ReleaseFacade {
        private final Validator validator=new Validator(); private final Deployer deployer=new Deployer(); private final Verifier verifier=new Verifier();
        Map<String,Object> release(Map<String,Object> request){
            String artifact=String.valueOf(request.getOrDefault("artifact","")); List<String> steps=new ArrayList<>(); steps.add(validator.validate(artifact));
            Map<String,Object> result=new LinkedHashMap<>(); result.put("artifact",artifact);
            if(Boolean.TRUE.equals(request.get("dry_run"))){steps.add("plan:"+artifact);result.put("status","planned");result.put("steps",steps);return result;}
            steps.add(deployer.deploy(artifact));steps.add(verifier.verify(artifact,Boolean.TRUE.equals(request.get("healthy"))));result.put("status","released");result.put("steps",steps);return result;
        }
    }
    @SuppressWarnings("unchecked") static Object evaluate(Map<String,Object> input){ReleaseFacade facade=new ReleaseFacade();List<Map<String,Object>> out=new ArrayList<>();for(Map<String,Object> item:(List<Map<String,Object>>)input.getOrDefault("releases",List.of()))out.add(facade.release(item));return Map.of("releases",out);}
}
