import java.util.ArrayList; import java.util.LinkedHashMap; import java.util.List; import java.util.Map;
final class BuilderPattern {
  static final class PatternException extends RuntimeException{private final String code;PatternException(String c){super(c);code=c;}String code(){return code;}}
  record ChatRequest(String model,List<String> messages,int maxTokens,boolean stream){}
  static final class ChatRequestBuilder{String model="";List<String> messages=new ArrayList<>();int maxTokens;boolean stream;
    @SuppressWarnings("unchecked") ChatRequestBuilder configure(Map<String,Object> v){model=String.valueOf(v.getOrDefault("model",""));messages=new ArrayList<>((List<String>)v.getOrDefault("messages",List.of()));maxTokens=((Number)v.getOrDefault("max_tokens",0)).intValue();stream=Boolean.TRUE.equals(v.get("stream"));return this;}
    ChatRequest build(){if(model.isEmpty()||messages.isEmpty()||maxTokens<=0)throw new PatternException("invalid_request");ChatRequest r=new ChatRequest(model,List.copyOf(messages),maxTokens,stream);model="";messages=new ArrayList<>();maxTokens=0;stream=false;return r;}}
  @SuppressWarnings("unchecked") static Object evaluate(Map<String,Object> input){ChatRequestBuilder b=new ChatRequestBuilder();List<Map<String,Object>> out=new ArrayList<>();for(Map<String,Object> v:(List<Map<String,Object>>)input.getOrDefault("builds",List.of())){ChatRequest r=b.configure(v).build();Map<String,Object> item=new LinkedHashMap<>();item.put("model",r.model());item.put("messages",r.messages());item.put("max_tokens",r.maxTokens());item.put("stream",r.stream());out.add(item);}return Map.of("requests",out);}
}
