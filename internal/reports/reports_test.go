package reports

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

func sampleComplianceData() []storage.ComplianceResult {
	v1 := 7.2
	v2 := 15.0
	return []storage.ComplianceResult{
		{
			FacilityName:  "North Water Treatment Plant",
			LocationName:  "Effluent",
			ParameterCode: "PH",
			ParameterName: "pH",
			ResultValue:   &v1,
			UnitCode:      "SU",
			CollectedAt:   time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
			Status:        "approved",
			LimitType:     "daily_max",
			LimitValue:    9.0,
			Compliance:    "OK",
		},
		{
			FacilityName:  "North Water Treatment Plant",
			LocationName:  "Effluent",
			ParameterCode: "TURB",
			ParameterName: "Turbidity",
			ResultValue:   &v2,
			UnitCode:      "NTU",
			CollectedAt:   time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
			Status:        "approved",
			LimitType:     "daily_max",
			LimitValue:    10.0,
			Compliance:    "EXCEEDANCE",
		},
	}
}

// =========================================================================
// Excel
// =========================================================================

func TestWriteComplianceExcel(t *testing.T) {
	var buf bytes.Buffer
	err := WriteComplianceExcel(&buf, "Test Facility", sampleComplianceData())
	if err != nil {
		t.Fatalf("WriteComplianceExcel: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("Excel output should not be empty")
	}
	// XLSX files start with PK (zip magic bytes)
	if buf.Bytes()[0] != 'P' || buf.Bytes()[1] != 'K' {
		t.Error("output should be a valid XLSX (zip) file")
	}
}

func TestWriteComplianceExcelEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteComplianceExcel(&buf, "Empty Facility", nil)
	if err != nil {
		t.Fatalf("WriteComplianceExcel with empty data: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("Excel output should not be empty even with no data")
	}
}

func TestWriteComplianceExcelWithQualifier(t *testing.T) {
	q := "<"
	data := []storage.ComplianceResult{
		{
			FacilityName:  "F",
			LocationName:  "L",
			ParameterName: "Ammonia",
			Qualifier:     &q,
			UnitCode:      "mg/L",
			CollectedAt:   time.Now(),
			LimitType:     "daily_max",
			LimitValue:    5.0,
			Compliance:    "OK",
		},
	}
	var buf bytes.Buffer
	if err := WriteComplianceExcel(&buf, "Qualifier Test", data); err != nil {
		t.Fatalf("WriteComplianceExcel with qualifier: %v", err)
	}
}

func TestWriteComplianceExcelLongFacilityName(t *testing.T) {
	longName := strings.Repeat("A Very Long Facility Name ", 10)
	var buf bytes.Buffer
	err := WriteComplianceExcel(&buf, longName, sampleComplianceData())
	if err != nil {
		t.Fatalf("WriteComplianceExcel with long name: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("should produce valid output")
	}
}

func TestWriteComplianceExcelLargeDataset(t *testing.T) {
	v := 7.0
	data := make([]storage.ComplianceResult, 1000)
	for i := range data {
		data[i] = storage.ComplianceResult{
			FacilityName:  "F",
			LocationName:  "L",
			ParameterCode: "PH",
			ParameterName: "pH",
			ResultValue:   &v,
			UnitCode:      "SU",
			CollectedAt:   time.Now(),
			LimitType:     "daily_max",
			LimitValue:    9.0,
			Compliance:    "OK",
		}
	}
	var buf bytes.Buffer
	if err := WriteComplianceExcel(&buf, "Large", data); err != nil {
		t.Fatalf("WriteComplianceExcel with 1000 rows: %v", err)
	}
}

func TestWriteComplianceExcelNilValueNilQualifier(t *testing.T) {
	data := []storage.ComplianceResult{
		{
			FacilityName:  "F",
			LocationName:  "L",
			ParameterName: "P",
			ResultValue:   nil,
			Qualifier:     nil,
			UnitCode:      "mg/L",
			CollectedAt:   time.Now(),
			LimitType:     "daily_max",
			LimitValue:    10,
			Compliance:    "N/A",
		},
	}
	var buf bytes.Buffer
	if err := WriteComplianceExcel(&buf, "NilVal", data); err != nil {
		t.Fatalf("WriteComplianceExcel nil value+qualifier: %v", err)
	}
}

// =========================================================================
// PDF
// =========================================================================

func TestWriteCompliancePDF(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCompliancePDF(&buf, "Test Facility", sampleComplianceData())
	if err != nil {
		t.Fatalf("WriteCompliancePDF: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("PDF output should not be empty")
	}
	if string(buf.Bytes()[:4]) != "%PDF" {
		t.Error("output should be a valid PDF file")
	}
}

func TestWriteCompliancePDFEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCompliancePDF(&buf, "Empty Facility", nil)
	if err != nil {
		t.Fatalf("WriteCompliancePDF with empty data: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("PDF output should not be empty even with no data")
	}
}

func TestWriteCompliancePDFCountsSummary(t *testing.T) {
	v := 5.0
	data := []storage.ComplianceResult{
		{FacilityName: "F", LocationName: "L", ParameterName: "P", ResultValue: &v, UnitCode: "mg/L", CollectedAt: time.Now(), LimitType: "daily_max", LimitValue: 10, Compliance: "OK"},
		{FacilityName: "F", LocationName: "L", ParameterName: "P", ResultValue: &v, UnitCode: "mg/L", CollectedAt: time.Now(), LimitType: "daily_max", LimitValue: 3, Compliance: "EXCEEDANCE"},
		{FacilityName: "F", LocationName: "L", ParameterName: "P", UnitCode: "mg/L", CollectedAt: time.Now(), LimitType: "daily_max", LimitValue: 10, Compliance: "N/A"},
	}
	var buf bytes.Buffer
	if err := WriteCompliancePDF(&buf, "Multi Status", data); err != nil {
		t.Fatalf("WriteCompliancePDF: %v", err)
	}
}

func TestWriteCompliancePDFSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCompliancePDF(&buf, "Facility & <Plant> \"North\" #1", sampleComplianceData())
	if err != nil {
		t.Fatalf("WriteCompliancePDF with special chars: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("should produce valid output")
	}
}

func TestWriteCompliancePDFLargeDataset(t *testing.T) {
	v := 7.0
	data := make([]storage.ComplianceResult, 1000)
	for i := range data {
		data[i] = storage.ComplianceResult{
			FacilityName:  "F",
			LocationName:  "L",
			ParameterCode: "PH",
			ParameterName: "pH",
			ResultValue:   &v,
			UnitCode:      "SU",
			CollectedAt:   time.Now(),
			LimitType:     "daily_max",
			LimitValue:    9.0,
			Compliance:    "OK",
		}
	}
	var buf bytes.Buffer
	if err := WriteCompliancePDF(&buf, "Large", data); err != nil {
		t.Fatalf("WriteCompliancePDF with 1000 rows: %v", err)
	}
}

func TestWriteCompliancePDFNilValueNilQualifier(t *testing.T) {
	data := []storage.ComplianceResult{
		{
			FacilityName:  "F",
			LocationName:  "L",
			ParameterName: "P",
			ResultValue:   nil,
			Qualifier:     nil,
			UnitCode:      "mg/L",
			CollectedAt:   time.Now(),
			LimitType:     "daily_max",
			LimitValue:    10,
			Compliance:    "N/A",
		},
	}
	var buf bytes.Buffer
	if err := WriteCompliancePDF(&buf, "NilVal", data); err != nil {
		t.Fatalf("WriteCompliancePDF nil value+qualifier: %v", err)
	}
}

// =========================================================================
// formatLimitType
// =========================================================================

func TestFormatLimitType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"daily_max", "Daily Max"},
		{"daily_min", "Daily Min"},
		{"monthly_avg", "Monthly Avg"},
		{"weekly_avg", "Weekly Avg"},
		{"instantaneous_max", "Inst. Max"},
		{"unknown_type", "unknown_type"},
	}
	for _, tt := range tests {
		got := formatLimitType(tt.input)
		if got != tt.want {
			t.Errorf("formatLimitType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
