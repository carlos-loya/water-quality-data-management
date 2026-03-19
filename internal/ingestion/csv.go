package ingestion

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// CSVImportResult summarizes the outcome of a CSV import.
type CSVImportResult struct {
	TotalRows    int            `json:"total_rows"`
	Imported     int            `json:"imported"`
	Rejected     int            `json:"rejected"`
	Results      []storage.SampleResult `json:"results,omitempty"`
	Errors       []RowError     `json:"errors,omitempty"`
}

// RowError describes why a specific CSV row was rejected.
type RowError struct {
	Row    int    `json:"row"`
	Field  string `json:"field"`
	Detail string `json:"detail"`
}

// Required CSV columns.
var requiredColumns = []string{
	"monitoring_location",
	"parameter_code",
	"collected_at",
	"result_value",
	"unit_code",
}

// CSVImporter handles parsing and importing CSV data into sample_results.
type CSVImporter struct {
	queries *storage.Queries
}

// NewCSVImporter creates a new importer.
func NewCSVImporter(queries *storage.Queries) *CSVImporter {
	return &CSVImporter{queries: queries}
}

// Import reads a CSV from r and inserts valid rows as sample results.
// orgID scopes the lookup of parameters, units, and monitoring locations.
// enteredBy is the user performing the import.
func (imp *CSVImporter) Import(ctx context.Context, r io.Reader, orgID, enteredBy uuid.UUID) (*CSVImportResult, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	colIndex := buildColumnIndex(header)

	// Validate required columns exist
	for _, col := range requiredColumns {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	// Build lookup maps for validation
	locations, err := imp.queries.ListAllMonitoringLocations(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("load locations: %w", err)
	}
	locByName := make(map[string]storage.MonitoringLocation, len(locations))
	for _, l := range locations {
		locByName[strings.ToUpper(l.Name)] = l
	}

	parameters, err := imp.queries.ListParameters(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("load parameters: %w", err)
	}
	paramByCode := make(map[string]storage.Parameter, len(parameters))
	for _, p := range parameters {
		paramByCode[strings.ToUpper(p.Code)] = p
	}

	units, err := imp.queries.ListUnits(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("load units: %w", err)
	}
	unitByCode := make(map[string]uuid.UUID, len(units))
	for _, u := range units {
		unitByCode[strings.ToUpper(u.Code)] = u.ID
	}

	// Process rows
	result := &CSVImportResult{}
	rowNum := 1 // 1-indexed, header is row 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row %d: %w", rowNum+1, err)
		}
		rowNum++
		result.TotalRows++

		params, rowErrors := imp.parseRow(record, colIndex, rowNum, locByName, paramByCode, unitByCode, enteredBy)
		if len(rowErrors) > 0 {
			result.Errors = append(result.Errors, rowErrors...)
			result.Rejected++
			continue
		}

		sr, err := imp.queries.CreateSampleResult(ctx, params)
		if err != nil {
			result.Errors = append(result.Errors, RowError{
				Row:    rowNum,
				Field:  "",
				Detail: fmt.Sprintf("database insert: %v", err),
			})
			result.Rejected++
			continue
		}
		result.Results = append(result.Results, sr)
		result.Imported++
	}

	return result, nil
}

func (imp *CSVImporter) parseRow(
	record []string,
	colIndex map[string]int,
	rowNum int,
	locByName map[string]storage.MonitoringLocation,
	paramByCode map[string]storage.Parameter,
	unitByCode map[string]uuid.UUID,
	enteredBy uuid.UUID,
) (storage.CreateSampleResultParams, []RowError) {
	var errors []RowError
	var params storage.CreateSampleResultParams

	params.EnteredBy = enteredBy
	params.Source = "csv_import"

	// Monitoring location
	locName := strings.TrimSpace(getCol(record, colIndex, "monitoring_location"))
	loc, ok := locByName[strings.ToUpper(locName)]
	if !ok || locName == "" {
		errors = append(errors, RowError{rowNum, "monitoring_location", fmt.Sprintf("unknown location: %q", locName)})
	} else {
		params.MonitoringLocationID = loc.ID
	}

	// Parameter
	paramCode := strings.TrimSpace(getCol(record, colIndex, "parameter_code"))
	param, ok := paramByCode[strings.ToUpper(paramCode)]
	if !ok || paramCode == "" {
		errors = append(errors, RowError{rowNum, "parameter_code", fmt.Sprintf("unknown parameter: %q", paramCode)})
	} else {
		params.ParameterID = param.ID
	}

	// Unit
	unitCode := strings.TrimSpace(getCol(record, colIndex, "unit_code"))
	unitID, ok := unitByCode[strings.ToUpper(unitCode)]
	if !ok || unitCode == "" {
		errors = append(errors, RowError{rowNum, "unit_code", fmt.Sprintf("unknown unit: %q", unitCode)})
	} else {
		params.UnitID = unitID
	}

	// Collected at
	collectedStr := strings.TrimSpace(getCol(record, colIndex, "collected_at"))
	collected, err := parseTimestamp(collectedStr)
	if err != nil {
		errors = append(errors, RowError{rowNum, "collected_at", fmt.Sprintf("invalid timestamp: %q", collectedStr)})
	} else {
		params.CollectedAt = collected
	}

	// Result value — may include qualifier prefix like "<0.1"
	valueStr := strings.TrimSpace(getCol(record, colIndex, "result_value"))
	if valueStr == "" {
		errors = append(errors, RowError{rowNum, "result_value", "empty"})
	} else if strings.HasPrefix(valueStr, "<") || strings.EqualFold(valueStr, "ND") {
		qualifier := "<"
		if strings.EqualFold(valueStr, "ND") {
			qualifier = "ND"
		}
		params.ResultQualifier = &qualifier
		// Try to parse detection limit from "<0.1" format
		if strings.HasPrefix(valueStr, "<") {
			numStr := strings.TrimPrefix(valueStr, "<")
			if dl, err := strconv.ParseFloat(numStr, 64); err == nil {
				params.DetectionLimit = &dl
			}
		}
	} else {
		val, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			errors = append(errors, RowError{rowNum, "result_value", fmt.Sprintf("not a number: %q", valueStr)})
		} else {
			params.ResultValue = &val
		}
	}

	// Optional: notes
	if idx, ok := colIndex["notes"]; ok && idx < len(record) {
		note := strings.TrimSpace(record[idx])
		if note != "" {
			params.Notes = &note
		}
	}

	return params, errors
}

func buildColumnIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, col := range header {
		idx[strings.ToLower(strings.TrimSpace(col))] = i
	}
	return idx
}

func getCol(record []string, colIndex map[string]int, name string) string {
	i, ok := colIndex[name]
	if !ok || i >= len(record) {
		return ""
	}
	return record[i]
}

func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"01/02/2006 15:04",
		"01/02/2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized format")
}
