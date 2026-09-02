export class PatternError extends Error { constructor(code) { super(code); this.code = code; } }

export function evaluate(input) {
    if(input.worker_count<1)throw new PatternError("invalid_worker_count");const available=Array(input.worker_count).fill(0),used=new Set(),executions=[];
    for(const job of input.jobs??[]){if(job.duration<1)throw new PatternError("invalid_duration");if(!["success","failure"].includes(job.outcome))throw new PatternError("unknown_job_outcome");let worker=0;for(let index=1;index<available.length;index++)if(available[index]<available[worker])worker=index;const start=available[worker],finish=start+job.duration;available[worker]=finish;used.add(worker);executions.push({id:job.id,worker,start,finish,status:job.outcome==="success"?"completed":"failed"});}
    return{executions,workers_used:used.size,makespan:Math.max(0,...available),joined:true};
}
