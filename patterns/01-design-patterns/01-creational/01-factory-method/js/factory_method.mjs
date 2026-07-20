export class PatternError extends Error {
  constructor(code) { super(code); this.code = code; }
}

class CsvFormatter {
  get mediaType() { return "text/csv"; }
  render(records) {
    return ["id,name", ...records.map((record) => `${record.id},${record.name}`)].join("\n");
  }
}

class JsonFormatter {
  get mediaType() { return "application/json"; }
  render(records) { return records.map((record) => ({ ...record })); }
}

class ExportJob {
  createFormatter() { throw new Error("abstract factory method"); }
  export(records) {
    const formatter = this.createFormatter();
    return { media_type: formatter.mediaType, body: formatter.render(records) };
  }
}

class CsvExportJob extends ExportJob { createFormatter() { return new CsvFormatter(); } }
class JsonExportJob extends ExportJob { createFormatter() { return new JsonFormatter(); } }

export function evaluate(input) {
  const creators = { csv: CsvExportJob, json: JsonExportJob };
  const Creator = creators[input.format];
  if (Creator === undefined) throw new PatternError("unsupported_format");
  return new Creator().export(input.records ?? []);
}
