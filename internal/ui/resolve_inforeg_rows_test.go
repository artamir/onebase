package ui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// Issue #44: в списке регистра сведений UUID ссылочных измерений/ресурсов
// должны заменяться на наименования, как это происходит в регистрах накопления.
func TestResolveInfoRegRows_RefDimension(t *testing.T) {
	ctx := context.Background()
	db, err := storage.ConnectSQLite(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Справочник-измерение
	kontragent := &metadata.Entity{
		Name:   "Контрагент",
		Kind:   metadata.KindCatalog,
		Fields: []metadata.Field{{Name: "Наименование", Type: metadata.FieldTypeString}},
	}
	if err := db.Migrate(ctx, []*metadata.Entity{kontragent}); err != nil {
		t.Fatal(err)
	}
	kID := uuid.New()
	if err := db.Upsert(ctx, "Контрагент", kID, map[string]any{"Наименование": "ООО Ромашка"}, kontragent); err != nil {
		t.Fatal(err)
	}

	registry := runtime.NewRegistry()
	registry.Load(runtime.LoadOptions{Entities: []*metadata.Entity{kontragent}})
	s := &Server{store: db, reg: registry}

	ir := &metadata.InfoRegister{
		Name: "ЦеныКонтрагентов",
		Dimensions: []metadata.Field{
			{Name: "Контрагент", Type: "reference:Контрагент", RefEntity: "Контрагент"},
		},
		Resources: []metadata.Field{
			{Name: "Цена", Type: metadata.FieldTypeNumber},
		},
	}

	rows := []map[string]any{
		{"Контрагент": kID.String(), "Цена": "100"},
	}
	s.resolveInfoRegRows(ctx, rows, ir)

	if rows[0]["Контрагент"] != "ООО Ромашка" {
		t.Errorf("UUID измерения не резолвнулся: got %v, want 'ООО Ромашка'", rows[0]["Контрагент"])
	}
}
