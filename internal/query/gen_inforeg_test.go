package query_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

func TestGen_LatestPerKey_SQLite(t *testing.T) {
	src := `ВЫБРАТЬ Номенклатура, ТипЦен, Цена ИЗ РегистрСведений.ЦеныНоменклатуры.СрезПоследних(&НаДату)`
	ir := &metadata.InfoRegister{
		Name: "ЦеныНоменклатуры", Periodic: true,
		Dimensions: []metadata.Field{
			{Name: "Номенклатура", Type: "reference:Номенклатура", RefEntity: "Номенклатура"},
			{Name: "ТипЦен", Type: "reference:ТипЦен", RefEntity: "ТипЦен"},
		},
		Resources: []metadata.Field{{Name: "Цена", Type: "number"}},
	}
	r, err := query.Compile(src, query.CompileOpts{
		InfoRegs: []*metadata.InfoRegister{ir},
		Dialect:  storage.SQLiteDialect{},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(r.SQL)
	// Должно использовать alias-имена в outer SELECT, не «номенклатура_id»
	if strings.Contains(r.SQL, "SELECT period, номенклатура_id AS номенклатура") {
		// inner — ОК, это нормально
	}
	// outer SELECT (после _w WHERE _rn = 1) — не должен содержать «_id»
	tail := r.SQL[strings.Index(r.SQL, "_w WHERE _rn = 1"):]
	if strings.Contains(tail, "номенклатура_id") || strings.Contains(tail, "типцен_id") {
		t.Errorf("outer SELECT всё ещё ссылается на *_id колонки:\n%s", r.SQL)
	}
}
