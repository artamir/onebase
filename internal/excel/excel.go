package excel

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportList builds an xlsx workbook from headers + rows and returns the raw bytes.
// Cells are formatted: dates → "ДД.ММ.ГГГГ", numbers → right-aligned.
func ExportList(cols []string, rows [][]any) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Лист1"
	f.SetSheetName("Sheet1", sheet)

	// Styles
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E2E8F0"}, Pattern: 1},
		Font: &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "CBD5E1", Style: 1},
			{Type: "right", Color: "CBD5E1", Style: 1},
			{Type: "top", Color: "CBD5E1", Style: 1},
			{Type: "bottom", Color: "CBD5E1", Style: 1},
		},
	})
	cellStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: false},
		Border: []excelize.Border{
			{Type: "left", Color: "E2E8F0", Style: 1},
			{Type: "right", Color: "E2E8F0", Style: 1},
			{Type: "top", Color: "E2E8F0", Style: 1},
			{Type: "bottom", Color: "E2E8F0", Style: 1},
		},
	})
	numStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "top"},
		Border: []excelize.Border{
			{Type: "left", Color: "E2E8F0", Style: 1},
			{Type: "right", Color: "E2E8F0", Style: 1},
			{Type: "top", Color: "E2E8F0", Style: 1},
			{Type: "bottom", Color: "E2E8F0", Style: 1},
		},
	})

	// Header row
	for ci, col := range cols {
		cell, _ := excelize.CoordinatesToCellName(ci+1, 1)
		f.SetCellValue(sheet, cell, col)
		f.SetCellStyle(sheet, cell, cell, boldStyle)
	}
	f.SetRowHeight(sheet, 1, 22)

	// Freeze header row
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Data rows
	for ri, row := range rows {
		rowIdx := ri + 2
		for ci, val := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, rowIdx)
			switch v := val.(type) {
			case time.Time:
				f.SetCellValue(sheet, cell, v.Format("02.01.2006"))
				f.SetCellStyle(sheet, cell, cell, cellStyle)
			case float64, float32, int, int32, int64, uint, uint32, uint64:
				f.SetCellValue(sheet, cell, v)
				f.SetCellStyle(sheet, cell, cell, numStyle)
			case nil:
				f.SetCellValue(sheet, cell, "")
				f.SetCellStyle(sheet, cell, cell, cellStyle)
			default:
				f.SetCellValue(sheet, cell, fmt.Sprintf("%v", v))
				f.SetCellStyle(sheet, cell, cell, cellStyle)
			}
		}
		f.SetRowHeight(sheet, rowIdx, 18)
	}

	// Auto column width (approximate: max 40 chars)
	for ci, col := range cols {
		colLetter, _ := excelize.ColumnNumberToName(ci + 1)
		maxLen := len(col)
		for _, row := range rows {
			if ci < len(row) {
				s := fmt.Sprintf("%v", row[ci])
				if len(s) > maxLen {
					maxLen = len(s)
				}
			}
		}
		w := float64(maxLen) * 1.1
		if w < 8 {
			w = 8
		}
		if w > 40 {
			w = 40
		}
		f.SetColWidth(sheet, colLetter, colLetter, w)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
