package interpreter

import (
	"testing"

	"github.com/ivantit66/onebase/internal/printform"
)

// TestMaketGetArea_NewFields ensures the new LayoutCell properties
// (VAlign, FontFamily, Border, RowSpan) are propagated to SpreadsheetDocumentCell
// when an area is materialised via макет.ПолучитьОбласть.
func TestMaketGetArea_NewFields(t *testing.T) {
	lt := &printform.LayoutTemplate{
		Name: "T",
		Areas: map[string]*printform.LayoutArea{
			"Шапка": {
				Rows: []printform.LayoutRow{
					{Cells: []printform.LayoutCell{
						{
							Text:       "Hello",
							VAlign:     "Middle",
							FontFamily: "Arial",
							Border:     "Thick",
							RowSpan:    2,
							ColSpan:    3,
						},
					}},
				},
			},
		},
	}

	m := NewMaket(lt)
	area := m.getArea("Шапка")
	if area == nil {
		t.Fatal("area is nil")
	}
	cell, ok := area.cells["0,0"]
	if !ok {
		t.Fatalf("cell 0,0 missing in area: %#v", area.cells)
	}
	if cell.vertical != "middle" {
		t.Errorf("vertical: got %q, want middle", cell.vertical)
	}
	if cell.fontFamily != "Arial" {
		t.Errorf("fontFamily: got %q, want Arial", cell.fontFamily)
	}
	if cell.border != "thick" {
		t.Errorf("border: got %q, want thick", cell.border)
	}
	if cell.rowSpan != 2 {
		t.Errorf("rowSpan: got %d, want 2", cell.rowSpan)
	}
	if cell.colSpan != 3 {
		t.Errorf("colSpan: got %d, want 3", cell.colSpan)
	}
}

// TestMaketGetArea_DefaultsPreserved ensures old layouts without new fields
// still produce cells with sensible defaults.
func TestMaketGetArea_DefaultsPreserved(t *testing.T) {
	lt := &printform.LayoutTemplate{
		Areas: map[string]*printform.LayoutArea{
			"A": {Rows: []printform.LayoutRow{
				{Cells: []printform.LayoutCell{{Text: "x"}}},
			}},
		},
	}
	area := NewMaket(lt).getArea("A")
	cell := area.cells["0,0"]
	if cell.vertical != "top" || cell.fontFamily != "Times New Roman" || cell.border != "all" {
		t.Errorf("defaults overwritten unexpectedly: %+v", cell)
	}
}
