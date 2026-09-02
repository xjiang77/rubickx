export class PatternError extends Error{constructor(code){super(code);this.code=code;}}
let instance;
class ProviderCatalog{constructor(){if(instance!==undefined)return instance;this.entriesMap=new Map();instance=this;}register(name,endpoint){const current=this.entriesMap.get(name);if(current!==undefined&&current!==endpoint)throw new PatternError("registration_conflict");this.entriesMap.set(name,endpoint);}entries(){return Object.fromEntries([...this.entriesMap.entries()].sort(([a],[b])=>a.localeCompare(b)));}}
const resetForTest=()=>{instance=undefined;};
export function evaluate(input){resetForTest();const first=new ProviderCatalog(),second=new ProviderCatalog();for(const value of input.registrations??[])first.register(value.name,value.endpoint);const entries=second.entries();return{same_instance:first===second,entries,size:Object.keys(entries).length};}
