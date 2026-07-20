export class PatternError extends Error{constructor(code){super(code);this.code=code;}}
class SupportMediator{routes={customer:["agent","bot"],agent:["customer"],bot:["agent"]};dispatch(event){const recipients=this.routes[event.from];if(recipients===undefined)throw new PatternError("unsupported_sender");return recipients.map((target)=>({from:event.from,to:target,message:event.message}));}}
export function evaluate(input){const mediator=new SupportMediator();return{deliveries:(input.events??[]).flatMap((event)=>mediator.dispatch(event))};}
