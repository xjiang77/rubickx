package factorymethod

import (
	"fmt"
	"strings"
)

type PatternError struct{ code string }

func (e *PatternError) Error() string { return e.code }
func (e *PatternError) Code() string  { return e.code }

type Formatter interface {
	MediaType() string
	Render([]map[string]string) any
}

type csvFormatter struct{}

func (csvFormatter) MediaType() string { return "text/csv" }
func (csvFormatter) Render(records []map[string]string) any {
	rows := []string{"id,name"}
	for _, record := range records {
		rows = append(rows, fmt.Sprintf("%s,%s", record["id"], record["name"]))
	}
	return strings.Join(rows, "\n")
}

type jsonFormatter struct{}

func (jsonFormatter) MediaType() string                      { return "application/json" }
func (jsonFormatter) Render(records []map[string]string) any { return records }

type ExportJob interface{ CreateFormatter() Formatter }
type csvExportJob struct{}

func (csvExportJob) CreateFormatter() Formatter { return csvFormatter{} }

type jsonExportJob struct{}

func (jsonExportJob) CreateFormatter() Formatter { return jsonFormatter{} }

func export(job ExportJob, records []map[string]string) map[string]any {
	formatter := job.CreateFormatter()
	return map[string]any{"media_type": formatter.MediaType(), "body": formatter.Render(records)}
}

func Evaluate(input map[string]any) (any, error) {
	rawRecords, _ := input["records"].([]any)
	records := make([]map[string]string, 0, len(rawRecords))
	for _, raw := range rawRecords {
		item := raw.(map[string]any)
		records = append(records, map[string]string{"id": item["id"].(string), "name": item["name"].(string)})
	}
	switch input["format"] {
	case "csv":
		return export(csvExportJob{}, records), nil
	case "json":
		return export(jsonExportJob{}, records), nil
	default:
		return nil, &PatternError{code: "unsupported_format"}
	}
}
