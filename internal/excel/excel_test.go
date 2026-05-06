package excel

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestExportList(t *testing.T) {
	cols := []string{"Наименование", "Цена", "Дата"}
	rows := [][]any{
		{"Яблоко", 15.5, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		{"Банан", 20.0, time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)},
	}

	data, err := ExportList(cols, rows)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Parse the resulting xlsx and verify content
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)

	sheet := "Лист1"
	// Check headers
	a1, _ := f.GetCellValue(sheet, "A1")
	b1, _ := f.GetCellValue(sheet, "B1")
	require.Equal(t, "Наименование", a1)
	require.Equal(t, "Цена", b1)

	// Check first data row
	a2, _ := f.GetCellValue(sheet, "A2")
	require.Equal(t, "Яблоко", a2)

	// Check date formatting
	c2, _ := f.GetCellValue(sheet, "C2")
	require.Equal(t, "01.05.2026", c2)
}

func TestExportListEmpty(t *testing.T) {
	data, err := ExportList([]string{"A", "B"}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, data)
}
