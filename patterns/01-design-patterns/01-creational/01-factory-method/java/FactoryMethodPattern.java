import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

final class FactoryMethodPattern {
    private FactoryMethodPattern() {}

    static final class PatternException extends RuntimeException {
        private final String code;
        PatternException(String code) { super(code); this.code = code; }
        String code() { return code; }
    }

    interface Formatter {
        String mediaType();
        Object render(List<Map<String, Object>> records);
    }

    static final class CsvFormatter implements Formatter {
        public String mediaType() { return "text/csv"; }
        public Object render(List<Map<String, Object>> records) {
            List<String> rows = new ArrayList<>();
            rows.add("id,name");
            for (Map<String, Object> record : records) {
                rows.add(record.get("id") + "," + record.get("name"));
            }
            return String.join("\n", rows);
        }
    }

    static final class JsonFormatter implements Formatter {
        public String mediaType() { return "application/json"; }
        public Object render(List<Map<String, Object>> records) {
            return records.stream().map(LinkedHashMap::new).toList();
        }
    }

    abstract static class ExportJob {
        abstract Formatter createFormatter();
        Map<String, Object> export(List<Map<String, Object>> records) {
            Formatter formatter = createFormatter();
            Map<String, Object> result = new LinkedHashMap<>();
            result.put("media_type", formatter.mediaType());
            result.put("body", formatter.render(records));
            return result;
        }
    }

    static final class CsvExportJob extends ExportJob {
        Formatter createFormatter() { return new CsvFormatter(); }
    }
    static final class JsonExportJob extends ExportJob {
        Formatter createFormatter() { return new JsonFormatter(); }
    }

    @SuppressWarnings("unchecked")
    static Object evaluate(Map<String, Object> input) {
        ExportJob job = switch (String.valueOf(input.get("format"))) {
            case "csv" -> new CsvExportJob();
            case "json" -> new JsonExportJob();
            default -> throw new PatternException("unsupported_format");
        };
        return job.export((List<Map<String, Object>>) input.getOrDefault("records", List.of()));
    }
}
