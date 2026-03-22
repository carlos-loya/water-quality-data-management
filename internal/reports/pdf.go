package reports

import (
	"fmt"
	"io"
	"time"

	"github.com/jung-kurt/gofpdf/v2"

	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// WriteCompliancePDF generates a formatted PDF compliance report.
func WriteCompliancePDF(w io.Writer, facilityName string, results []storage.ComplianceResult) error {
	pdf := gofpdf.New("L", "mm", "Letter", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Title
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, fmt.Sprintf("%s - Compliance Report", facilityName), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006 3:04 PM")), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// Summary counts
	var ok, exceed, na int
	for _, r := range results {
		switch r.Compliance {
		case "OK":
			ok++
		case "EXCEEDANCE":
			exceed++
		default:
			na++
		}
	}
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 6, fmt.Sprintf("Total evaluations: %d    OK: %d    Exceedances: %d    N/A: %d", len(results), ok, exceed, na), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// Table header
	colWidths := []float64{35, 28, 45, 25, 20, 30, 25, 30}
	headers := []string{"Date", "Location", "Parameter", "Result", "Unit", "Limit Type", "Limit", "Compliance"}

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(31, 78, 121)
	pdf.SetTextColor(255, 255, 255)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Table rows
	pdf.SetFont("Helvetica", "", 8)
	for i, r := range results {
		// Alternating row background
		if i%2 == 0 {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		// Exceedance highlighting
		isExceed := r.Compliance == "EXCEEDANCE"
		if isExceed {
			pdf.SetFillColor(253, 232, 232)
			pdf.SetTextColor(204, 0, 0)
		} else {
			pdf.SetTextColor(0, 0, 0)
		}

		dateStr := r.CollectedAt.Format("01/02/2006 15:04")
		valStr := "N/A"
		if r.ResultValue != nil {
			valStr = fmt.Sprintf("%.2f", *r.ResultValue)
		} else if r.Qualifier != nil {
			valStr = *r.Qualifier
		}

		row := []string{
			dateStr,
			r.LocationName,
			r.ParameterName,
			valStr,
			r.UnitCode,
			formatLimitType(r.LimitType),
			fmt.Sprintf("%.2f", r.LimitValue),
			r.Compliance,
		}

		for j, cell := range row {
			align := "L"
			if j == 3 || j == 6 {
				align = "R"
			}
			if j == 7 {
				align = "C"
			}
			pdf.CellFormat(colWidths[j], 6, cell, "1", 0, align, true, 0, "")
		}
		pdf.Ln(-1)
	}

	// Footer
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 5, "This report was generated from the Water Quality Data Management system. Data is subject to review and approval workflows.", "", 1, "L", false, 0, "")

	return pdf.Output(w)
}

func formatLimitType(s string) string {
	switch s {
	case "daily_max":
		return "Daily Max"
	case "daily_min":
		return "Daily Min"
	case "monthly_avg":
		return "Monthly Avg"
	case "weekly_avg":
		return "Weekly Avg"
	case "instantaneous_max":
		return "Inst. Max"
	default:
		return s
	}
}
