import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

final class WorkerPoolPattern {
    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String,Object> input) {
        int count=((Number)input.get("worker_count")).intValue(); if(count<1)throw new PatternException("invalid_worker_count");
        int[] available=new int[count]; Set<Integer> used=new HashSet<>(); List<Map<String,Object>> executions=new ArrayList<>();
        for(Map<String,Object> job:(List<Map<String,Object>>)input.getOrDefault("jobs",List.of())){
            int duration=((Number)job.get("duration")).intValue(); if(duration<1)throw new PatternException("invalid_duration"); String outcome=String.valueOf(job.get("outcome")); if(!outcome.equals("success")&&!outcome.equals("failure"))throw new PatternException("unknown_job_outcome");
            int worker=0; for(int index=1;index<count;index++)if(available[index]<available[worker])worker=index; int start=available[worker],finish=start+duration;available[worker]=finish;used.add(worker);
            executions.add(Map.of("id",job.get("id"),"worker",worker,"start",start,"finish",finish,"status",outcome.equals("success")?"completed":"failed"));
        }
        int makespan=0;for(int value:available)makespan=Math.max(makespan,value);return Map.of("executions",executions,"workers_used",used.size(),"makespan",makespan,"joined",true);
    }
}
