import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class AbstractFactoryPattern {
    static final class PatternException extends RuntimeException { private final String code; PatternException(String code){super(code);this.code=code;} String code(){return code;} }
    interface CloudFactory { String family(); String queue(String prefix); String objectStore(String prefix); }
    static final class AwsFactory implements CloudFactory { public String family(){return "aws";} public String queue(String p){return "sqs:"+p;} public String objectStore(String p){return "s3:"+p;} }
    static final class GcpFactory implements CloudFactory { public String family(){return "gcp";} public String queue(String p){return "pubsub:"+p;} public String objectStore(String p){return "gcs:"+p;} }
    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String,Object> input) {
        String prefix = String.valueOf(input.get("prefix"));
        List<Map<String,Object>> resources = new ArrayList<>();
        for (Object raw : (List<Object>) input.getOrDefault("providers", List.of())) {
            CloudFactory factory = switch (String.valueOf(raw)) { case "aws" -> new AwsFactory(); case "gcp" -> new GcpFactory(); default -> throw new PatternException("unsupported_provider"); };
            Map<String,Object> value = new LinkedHashMap<>();
            value.put("family", factory.family()); value.put("queue", factory.queue(prefix)); value.put("object_store", factory.objectStore(prefix)); resources.add(value);
        }
        return Map.of("resources", resources);
    }
}
