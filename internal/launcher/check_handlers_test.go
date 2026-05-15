package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDSL_OK(t *testing.T) {
	src := `Процедура Привет()
    Сообщить("Hi")
КонецПроцедуры`
	if issues := parseDSL(src, "test.os"); len(issues) != 0 {
		t.Fatalf("expected clean parse, got %+v", issues)
	}
}

func TestParseDSL_Broken(t *testing.T) {
	src := `Процедура Привет(
    Сообщить("Hi")
КонецПроцедуры`
	issues := parseDSL(src, "test.os")
	if len(issues) == 0 {
		t.Fatal("expected at least one issue for missing )")
	}
}

func TestCheckWidgetYAML_OK(t *testing.T) {
	yaml := `name: ВыручкаМесяца
type: kpi
title: Выручка
format: money
query: ВЫБРАТЬ СУММА(Сумма) КАК Значение ИЗ Документ.X`
	if issues := checkWidgetYAML(yaml, "ВыручкаМесяца"); len(issues) != 0 {
		t.Fatalf("expected clean widget, got %+v", issues)
	}
}

func TestCheckWidgetYAML_UnknownType(t *testing.T) {
	yaml := `name: X
type: gauge
title: ok`
	issues := checkWidgetYAML(yaml, "X")
	if len(issues) == 0 {
		t.Fatal("expected error on unknown widget type")
	}
	if !strings.Contains(issues[0].Message, "type") && !strings.Contains(issues[0].Message, "тип") {
		t.Errorf("expected type-related message, got %q", issues[0].Message)
	}
}

func TestCheckHomePageYAML_Empty(t *testing.T) {
	if issues := checkHomePageYAML(""); len(issues) != 0 {
		t.Fatalf("empty body should be considered valid, got %+v", issues)
	}
}

func TestCheckHomePageYAML_Bad(t *testing.T) {
	bad := "title: Главная\nlayout: ::not-yaml::\n  - broken"
	if issues := checkHomePageYAML(bad); len(issues) == 0 {
		t.Fatal("expected YAML parse error")
	}
}

func TestCheckProjectDir(t *testing.T) {
	dir := t.TempDir()
	// good widget
	mkFile(t, filepath.Join(dir, "widgets", "ok.yaml"), `name: A
type: kpi
title: A
query: SELECT 1`)
	// broken DSL
	mkFile(t, filepath.Join(dir, "src", "broken.os"), `Процедура X(
КонецПроцедуры`)
	// good DSL
	mkFile(t, filepath.Join(dir, "src", "good.os"), `Процедура Y()
КонецПроцедуры`)

	issues := checkProjectDir(dir)
	var hasBroken bool
	for _, i := range issues {
		if strings.Contains(i.File, "broken.os") {
			hasBroken = true
		}
	}
	if !hasBroken {
		t.Fatalf("expected broken.os issue, got: %+v", issues)
	}
}

func mkFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
