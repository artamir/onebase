package launcher

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Регрессия: до фикса 2026-05-25 saveEntity содержала только Name/Posting/
// Fields/TableParts — все остальные ключи YAML (hierarchical, numerator,
// predefined, list_form, item_form, title, hierarchy_kind) ТЕРЯЛИСЬ при
// roundtrip Unmarshal → Marshal в configuratorSaveFields. После
// добавления «Поставщик» в Номенклатуру через UI у пользователя пропадал
// `hierarchical: true`, и интерфейс справочника терял группы/деревья.
//
// Этот тест защищает structural roundtrip: парсим типичный YAML, ничего
// не меняем, сериализуем обратно — все важные ключи должны сохраниться.
func TestSaveEntity_Roundtrip_PreservesAllFields(t *testing.T) {
	input := `name: Номенклатура
title: Каталог номенклатуры
hierarchical: true
hierarchy_kind: folders_and_items
list_form:
  - ФормаСписка
item_form:
  - ФормаОбъекта
predefined:
  - name: Услуги
    fields:
      Артикул: SERV
fields:
  - name: Наименование
    type: string
  - name: Поставщик
    type: reference:Контрагент
tableparts:
  - name: Цены
    fields:
      - name: Сумма
        type: number
`
	var ent saveEntity
	if err := yaml.Unmarshal([]byte(input), &ent); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := yaml.Marshal(&ent)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)

	for _, must := range []string{
		"name: Номенклатура",
		"title: Каталог номенклатуры",
		"hierarchical: true",
		"hierarchy_kind: folders_and_items",
		"list_form:",
		"item_form:",
		"predefined:",
		"- name: Услуги",
		"Артикул: SERV",
		"- name: Наименование",
		"- name: Поставщик",
		"tableparts:",
		"- name: Цены",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("после roundtrip потерян фрагмент %q\nполучилось:\n%s", must, got)
		}
	}
}

// applyFieldEdits: включение hierarchical=true должно прописать YAML-ключ
// независимо от исходного состояния, а сброс — стереть и hierarchy_kind.
func TestApplyFieldEdits_HierarchicalToggle(t *testing.T) {
	ent := &saveEntity{
		Name:          "Контрагент",
		Hierarchical:  true,
		HierarchyKind: "folders_and_items",
		Fields:        []saveField{{Name: "Наименование", Type: "string"}},
	}
	off := false
	applyFieldEdits(ent, ent.Fields, nil, nil, &off)
	if ent.Hierarchical {
		t.Errorf("после off-toggle Hierarchical=true, ожидалось false")
	}
	if ent.HierarchyKind != "" {
		t.Errorf("hierarchy_kind=%q, при выключении должен очищаться", ent.HierarchyKind)
	}

	on := true
	applyFieldEdits(ent, ent.Fields, nil, nil, &on)
	if !ent.Hierarchical {
		t.Errorf("после on-toggle Hierarchical=false, ожидалось true")
	}
}

// nil-указатель означает «не трогать поле» — UI редактор справочника не
// отправляет hierarchical при редактировании документа и наоборот.
func TestApplyFieldEdits_NilPtrPreserves(t *testing.T) {
	ent := &saveEntity{
		Name:         "Номенклатура",
		Hierarchical: true,
		Posting:      false,
		Fields:       []saveField{{Name: "X", Type: "string"}},
	}
	applyFieldEdits(ent, ent.Fields, nil, nil, nil)
	if !ent.Hierarchical {
		t.Errorf("nil hierarchical-ptr перетёр поле в false")
	}
	if ent.Posting {
		t.Errorf("nil posting-ptr изменил поле")
	}
}
