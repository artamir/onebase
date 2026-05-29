package interpreter

import (
	"path/filepath"
	"testing"
)

// Round-trip ЗаписьТекста → ЧтениеТекста: запись строк и их чтение обратно.
func TestTextWriterReader_RoundTrip(t *testing.T) {
	SetFileSandbox("")
	p := filepath.Join(t.TempDir(), "note.txt")

	w := &dslTextWriter{path: p}
	w.CallMethod("открыть", nil)
	w.CallMethod("записатьстроку", []any{"первая"})
	w.CallMethod("записать", []any{"вторая"})
	w.CallMethod("закрыть", nil)

	r := &dslTextReader{path: p}
	r.CallMethod("открыть", nil)
	if got := r.CallMethod("прочитатьстроку", nil); got != "первая" {
		t.Errorf("ПрочитатьСтроку(1): got %v", got)
	}
	if got := r.CallMethod("прочитатьстроку", nil); got != "вторая" {
		t.Errorf("ПрочитатьСтроку(2): got %v", got)
	}
	if got := r.CallMethod("прочитатьстроку", nil); got != nil {
		t.Errorf("ПрочитатьСтроку после конца: ожидался nil, got %v", got)
	}
}

// Файл.Существует отражает наличие файла.
func TestFileObject_Exists(t *testing.T) {
	SetFileSandbox("")
	p := filepath.Join(t.TempDir(), "x.txt")
	f := &dslFile{path: p}
	if got := f.Get("существует"); got != false {
		t.Errorf("несуществующий файл: ожидалось false, got %v", got)
	}

	w := &dslTextWriter{path: p}
	w.CallMethod("открыть", nil)
	w.CallMethod("записать", []any{"данные"})
	w.CallMethod("закрыть", nil)

	f2 := &dslFile{path: p}
	if got := f2.Get("существует"); got != true {
		t.Errorf("существующий файл: ожидалось true, got %v", got)
	}
}
