package reports

import (
	"fmt"
	"io"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// WriteComplianceExcel generates an Excel workbook with sample results and compliance evaluation.
func WriteComplianceExcel(w io.Writer, facilityName string, results []storage.ComplianceResult) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Compliance Report"
	f.SetSheetName("Sheet1", sheet)

	// Styles
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1F4E79"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	exceedStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "CC0000", Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FDE8E8"}},
	})
	dateStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 22, // m/d/yy h:mm
	})

	// Title row
	f.SetCellValue(sheet, "A1", fmt.Sprintf("%s — Compliance Report", facilityName))
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
	})
	f.SetCellStyle(sheet, "A1", "A1", titleStyle)
	f.MergeCell(sheet, "A1", "H1")

	f.SetCellValue(sheet, "A2", fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006 3:04 PM")))
	f.MergeCell(sheet, "A2", "H2")

	// Headers
	headers := []string{"Date", "Location", "Parameter", "Result", "Unit", "Limit Type", "Limit Value", "Compliance"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 4)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Data rows
	for i, r := range results {
		row := i + 5
		collected, _ := time.Parse(time.RFC3339, r.CollectedAt.Format(time.RFC3339))

		dateCell, _ := excelize.CoordinatesToCellName(1, row)
		f.SetCellValue(sheet, dateCell, collected)
		f.SetCellStyle(sheet, dateCell, dateCell, dateStyle)

		locCell, _ := excelize.CoordinatesToCellName(2, row)
		f.SetCellValue(sheet, locCell, r.LocationName)

		paramCell, _ := excelize.CoordinatesToCellName(3, row)
		f.SetCellValue(sheet, paramCell, r.ParameterName)

		valCell, _ := excelize.CoordinatesToCellName(4, row)
		if r.ResultValue != nil {
			f.SetCellValue(sheet, valCell, *r.ResultValue)
		} else if r.Qualifier != nil {
			f.SetCellValue(sheet, valCell, *r.Qualifier)
		} else {
			f.SetCellValue(sheet, valCell, "N/A")
		}

		unitCell, _ := excelize.CoordinatesToCellName(5, row)
		f.SetCellValue(sheet, unitCell, r.UnitCode)

		ltCell, _ := excelize.CoordinatesToCellName(6, row)
		f.SetCellValue(sheet, ltCell, r.LimitType)

		lvCell, _ := excelize.CoordinatesToCellName(7, row)
		f.SetCellValue(sheet, lvCell, r.LimitValue)

		compCell, _ := excelize.CoordinatesToCellName(8, row)
		f.SetCellValue(sheet, compCell, r.Compliance)

		if r.Compliance == "EXCEEDANCE" {
			startCell, _ := excelize.CoordinatesToCellName(1, row)
			endCell, _ := excelize.CoordinatesToCellName(8, row)
			f.SetCellStyle(sheet, startCell, endCell, exceedStyle)
		}
	}

	// Column widths
	widths := map[string]float64{"A": 18, "B": 14, "C": 22, "D": 12, "E": 12, "F": 14, "G": 12, "H": 14}
	for col, w := range widths {
		f.SetColWidth(sheet, col, col, w)
	}

	// Auto filter
	lastRow := len(results) + 4
	f.AutoFilter(sheet, fmt.Sprintf("A4:H%d", lastRow), nil)

	return f.Write(w)
}

// WriteSampleResultsExcel generates an Excel export of sample results.
func WriteSampleResultsExcel(w io.Writer, facilityName string, results []storage.ComplianceResult) error {
	// Reuses ComplianceResult since it already has the joined data we need.
	// For a raw export without compliance, we'd add a separate type.
	return WriteComplianceExcel(w, facilityName, results)
}
