export class PatternError extends Error { constructor(code) { super(code); this.code = code; } }

export function evaluate(input) {
    const changes=[], seen=new Set(); let writeBalance=0;
    for (const command of input.commands ?? []) {
        if (seen.has(command.id)) throw new PatternError("duplicate_command_id");
        seen.add(command.id); writeBalance += command.delta; changes.push({delta:command.delta});
    }
    let projectionBalance=0, projectionVersion=0; const snapshots=[];
    for (const target of input.projection_targets ?? []) {
        if (target < projectionVersion) throw new PatternError("projection_regression");
        if (target > changes.length) throw new PatternError("projection_ahead");
        for (const change of changes.slice(projectionVersion,target)) projectionBalance += change.delta;
        projectionVersion=target; snapshots.push({balance:projectionBalance,version:projectionVersion,lag:changes.length-projectionVersion});
    }
    return {write_model:{balance:writeBalance,version:changes.length},projection_snapshots:snapshots};
}
