export class PatternError extends Error{constructor(code){super(code);this.code=code;}}
class Equals{constructor(field,value){this.field=field;this.value=value;}evaluate(context,trace){trace.push(this.field);return context[this.field]===this.value;}}
class All{constructor(children){this.children=children;}evaluate(context,trace){for(const child of this.children)if(!child.evaluate(context,trace))return false;return true;}}
class AnyOf{constructor(children){this.children=children;}evaluate(context,trace){for(const child of this.children)if(child.evaluate(context,trace))return true;return false;}}
function build(value){if(value.type==="equals")return new Equals(value.field,value.value);if(value.type==="all"||value.type==="any"){const children=(value.children??[]).map(build);return value.type==="all"?new All(children):new AnyOf(children);}throw new PatternError("unsupported_node");}
export function evaluate(input){const trace=[];const allowed=build(input.tree).evaluate(input.context??{},trace);return{allowed,evaluated:trace};}
