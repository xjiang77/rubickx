export class PatternError extends Error{constructor(code){super(code);this.code=code;}}
class DocumentStore{constructor(){this.documents=new Map([["public","status:ok"],["secret","key:rotated"]]);}read(name){if(!this.documents.has(name))throw new PatternError("not_found");return this.documents.get(name);}}
class DocumentProxy{constructor(role){this.role=role;this.subject=undefined;this.loadCount=0;}read(name){if(name==="secret"&&this.role!=="admin")throw new PatternError("forbidden");if(this.subject===undefined){this.subject=new DocumentStore();this.loadCount++;}return this.subject.read(name);}}
export function evaluate(input){const proxy=new DocumentProxy(input.role??"viewer");const values=(input.reads??[]).map((name)=>proxy.read(name));return{values,load_count:proxy.loadCount};}
