package printform

import "testing"

func newTestCtx() *RenderContext {
	return &RenderContext{
		Document: map[string]any{
			"Номер":       "УПД-001",
			"Покупатель":  "ref-buyer",
			"Дата":        "2026-06-11",
		},
		Constants: map[string]any{
			"НазваниеОрганизации": "ООО Ромашка",
		},
		Refs: map[string]map[string]any{
			"ref-buyer": {"наименование": "ИП Иванов", "ИНН": "770101"},
			"ref-good":  {"наименование": "Стол"},
		},
		TableParts: map[string][]map[string]any{
			"Товары": {
				{"Номенклатура": "ref-good", "Количество": 2.0, "Сумма": 100.0},
				{"Номенклатура": "ref-good", "Количество": 3.0, "Сумма": 250.5},
			},
		},
	}
}

func TestResolveExprSimpleField(t *testing.T) {
	ctx := newTestCtx()
	if got := ResolveExpr("Номер", ctx, nil, 0); got != "УПД-001" {
		t.Fatalf("Номер = %v", got)
	}
}

func TestResolveExprRefDisplay(t *testing.T) {
	ctx := newTestCtx()
	// Покупатель — UUID-ссылка → должно вернуться наименование.
	if got := ResolveExpr("Покупатель", ctx, nil, 0); got != "ИП Иванов" {
		t.Fatalf("Покупатель = %v", got)
	}
}

func TestResolveExprSubField(t *testing.T) {
	ctx := newTestCtx()
	if got := ResolveExpr("Покупатель.ИНН", ctx, nil, 0); got != "770101" {
		t.Fatalf("Покупатель.ИНН = %v", got)
	}
}

func TestResolveExprConstant(t *testing.T) {
	ctx := newTestCtx()
	if got := ResolveExpr("Константы.НазваниеОрганизации", ctx, nil, 0); got != "ООО Ромашка" {
		t.Fatalf("constant = %v", got)
	}
}

func TestResolveExprRow(t *testing.T) {
	ctx := newTestCtx()
	row := ctx.TableParts["Товары"][0]
	if got := ResolveExpr("@row", ctx, row, 5); got != 5 {
		t.Fatalf("@row = %v", got)
	}
	if got := ResolveExpr("Номенклатура", ctx, row, 5); got != "Стол" {
		t.Fatalf("Номенклатура = %v", got)
	}
}

// TestResolveExprTotal verifies the new Итог.<ТЧ>.<Поле> aggregate.
func TestResolveExprTotal(t *testing.T) {
	ctx := newTestCtx()
	got := ResolveExpr("Итог.Товары.Сумма", ctx, nil, 0)
	f, ok := got.(float64)
	if !ok || f != 350.5 {
		t.Fatalf("Итог.Товары.Сумма = %v (want 350.5)", got)
	}
	// Case-insensitive table part name.
	got2 := ResolveExpr("Итог.товары.Количество", ctx, nil, 0)
	if f2, _ := got2.(float64); f2 != 5.0 {
		t.Fatalf("Итог.товары.Количество = %v (want 5)", got2)
	}
}

func TestResolveValueWithFormat(t *testing.T) {
	ctx := newTestCtx()
	if got := ResolveValue("Итог.Товары.Сумма | number:2", ctx, nil, 0); got != "350.50" {
		t.Fatalf("formatted total = %q (want 350.50)", got)
	}
	if got := ResolveValue("Дата | date", ctx, nil, 0); got != "11.06.2026" {
		t.Fatalf("formatted date = %q", got)
	}
}

func TestInterpolateText(t *testing.T) {
	ctx := newTestCtx()
	got := InterpolateText("Счёт № {{Номер}} для {{Покупатель}}", ctx, nil, 0)
	want := "Счёт № УПД-001 для ИП Иванов"
	if got != want {
		t.Fatalf("InterpolateText = %q (want %q)", got, want)
	}
}
