package query_test

import (
	"strings"
	"testing"

	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/query"
	"github.com/ivantit66/onebase/internal/storage"
)

// п.48: при авто-JOIN ссылочного поля собственная колонка основной таблицы
// должна квалифицироваться именем таблицы, иначе одноимённая колонка
// присоединённого каталога даёт ambiguous column.
func TestQualifyOwnColumn_UnderRefJoin(t *testing.T) {
	ent := &metadata.Entity{
		Name: "Заказ",
		Kind: metadata.KindDocument,
		Fields: []metadata.Field{
			{Name: "Контрагент", Type: "reference:Контрагент", RefEntity: "Контрагент"},
			{Name: "Статус", Type: metadata.FieldTypeString},
		},
	}
	src := `ВЫБРАТЬ Статус ИЗ Документ.Заказ ГДЕ Контрагент = &К`
	r, err := query.Compile(src, query.CompileOpts{
		Entities: []*metadata.Entity{ent},
		Params:   map[string]any{"К": "x"},
		Dialect:  storage.SQLiteDialect{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.SQL, "заказ.статус") {
		t.Errorf("ожидалась квалификация заказ.статус при ref-JOIN:\n%s", r.SQL)
	}
}

// Авто-JOIN при алиасе источника (КАК р): ON должен ссылаться на алиас, а не на
// сырое имя таблицы — иначе `no such column: таблица.col` (таблица доступна как р).
func TestRefJoin_AliasedSource(t *testing.T) {
	ent := &metadata.Entity{
		Name: "РеализацияТоваров",
		Kind: metadata.KindDocument,
		Fields: []metadata.Field{
			{Name: "Покупатель", Type: "reference:Контрагент", RefEntity: "Контрагент"},
		},
	}
	cp := &metadata.Entity{Name: "Контрагент", Kind: metadata.KindCatalog,
		Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}}}
	src := `ВЫБРАТЬ р.Покупатель ИЗ Документ.РеализацияТоваров КАК р`
	r, err := query.Compile(src, query.CompileOpts{
		Entities: []*metadata.Entity{ent, cp},
		Dialect:  storage.SQLiteDialect{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.SQL, "= р.покупатель_id") {
		t.Errorf("JOIN ON должен использовать алиас р, got:\n%s", r.SQL)
	}
	if strings.Contains(r.SQL, "реализациятоваров.покупатель_id") {
		t.Errorf("JOIN ON не должен ссылаться на сырое имя таблицы при алиасе:\n%s", r.SQL)
	}
}
