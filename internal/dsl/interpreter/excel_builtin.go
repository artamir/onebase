package interpreter

import (
	"encoding/base64"
	"fmt"

	"github.com/ivantit66/onebase/internal/excel"
)

func init() {
	builtins["ВыгрузитьВExcel"] = builtinExportExcel
	builtins["ExportExcel"] = builtinExportExcel
}

// builtinExportExcel(data, title)
// data — Массив массивов; первый подмассив — заголовки, остальные — строки данных.
// Возвращает base64-строку содержимого xlsx-файла.
func builtinExportExcel(args []any, file string, line int) (any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ВыгрузитьВExcel: ожидается аргумент Данные (Массив)")
	}

	outerArr, ok := args[0].(*Array)
	if !ok {
		return nil, fmt.Errorf("ВыгрузитьВExcel: аргумент Данные должен быть Массивом")
	}
	if len(outerArr.items) < 1 {
		return "", nil
	}

	// First row → column headers
	firstRow, ok := outerArr.items[0].(*Array)
	if !ok {
		return nil, fmt.Errorf("ВыгрузитьВExcel: первая строка (заголовки) должна быть Массивом")
	}
	cols := make([]string, len(firstRow.items))
	for i, v := range firstRow.items {
		cols[i] = fmt.Sprintf("%v", v)
	}

	// Data rows
	rows := make([][]any, 0, len(outerArr.items)-1)
	for _, rowVal := range outerArr.items[1:] {
		rowArr, ok := rowVal.(*Array)
		if !ok {
			continue
		}
		cells := make([]any, len(rowArr.items))
		copy(cells, rowArr.items)
		rows = append(rows, cells)
	}

	data, err := excel.ExportList(cols, rows)
	if err != nil {
		return nil, fmt.Errorf("ВыгрузитьВExcel: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
