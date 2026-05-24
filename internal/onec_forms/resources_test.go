package onec_forms

import (
	"os"
	"path/filepath"
	"testing"
)

// Создаёт фейковую структуру Items/<Name>/Picture.png|ValuesPicture.png,
// чтобы протестировать копирование без реальных бинарных файлов.
func setupItemsDir(t *testing.T, entries map[string]map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	for elementName, files := range entries {
		elDir := filepath.Join(dir, elementName)
		if err := os.MkdirAll(elDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, data := range files {
			if err := os.WriteFile(filepath.Join(elDir, name), data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

func TestCopyResources_Basic(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47} // png signature
	gifBytes := []byte{'G', 'I', 'F', '8'}

	itemsDir := setupItemsDir(t, map[string]map[string][]byte{
		"ЗелёнаяГалка": {
			"Picture.png": pngBytes,
		},
		"ТЗНоменклатурыПоле1": {
			"ValuesPicture.png": pngBytes,
			"Picture.gif":       gifBytes,
		},
		"СПустойПодпиской": {
			// пустая папка, должна быть проигнорирована
		},
		"СНеизвестнымФайлом": {
			"meta.xml": []byte("<xml/>"),
		},
	})

	dst := filepath.Join(t.TempDir(), "_resources")
	res, warns, err := CopyResources(itemsDir, dst)
	if err != nil {
		t.Fatal(err)
	}

	// ожидаем 3 файла (ЗелёнаяГалка/Picture.png, ТЗ.../ValuesPicture.png + Picture.gif)
	if len(res) != 3 {
		t.Errorf("resources count = %d, want 3", len(res))
		for _, r := range res {
			t.Logf("  %s", r.Path)
		}
	}

	// каждый файл реально создан
	for _, r := range res {
		// res.Path относителен к каталогу формы (parent от _resources)
		full := filepath.Join(filepath.Dir(dst), r.Path)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("файл не создан: %s — %v", full, err)
		}
	}

	// warning W050 для meta.xml
	var seenW050 bool
	for _, w := range warns {
		if w.Code == W050_NeedsReview && w.Field == "meta.xml" {
			seenW050 = true
		}
	}
	if !seenW050 {
		t.Error("ожидался warning W050 для meta.xml")
	}
}

func TestCopyResources_NoDir(t *testing.T) {
	res, warns, err := CopyResources(filepath.Join(t.TempDir(), "nope"), t.TempDir())
	if err != nil {
		t.Errorf("отсутствующий itemsDir должен вернуть nil-error: %v", err)
	}
	if res != nil || warns != nil {
		t.Errorf("res/warns должны быть nil: %v / %v", res, warns)
	}
}

func TestCopyResources_RealUT11(t *testing.T) {
	realItems := `C:\Projects\АА5БП3\УТ11УТ11\ПереносДанныхУТ11УТ11_52\Forms\Форма\Ext\Form\Items`
	if _, err := os.Stat(realItems); err != nil {
		t.Skip("real Items не найден")
	}
	dst := filepath.Join(t.TempDir(), "_resources")
	res, _, err := CopyResources(realItems, dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("реальная форма должна иметь хотя бы один ресурс")
	}
	t.Logf("copied %d resources from УТ11 form", len(res))
	for _, r := range res {
		t.Logf("  %s (%s)", r.Path, r.OriginalName)
	}
}

func TestAttachResourcesToForm(t *testing.T) {
	form := &IRForm{
		Elements: []*IRElement{
			{
				Name: "Группа",
				Kind: "UsualGroup",
				Children: []*IRElement{
					{Name: "ЗелёнаяГалка", Kind: "PictureField"},
					{Name: "ТЗНоменклатурыПоле1", Kind: "PictureField"},
					{Name: "БезРесурса", Kind: "InputField"},
				},
			},
		},
	}
	resources := []IRResource{
		{ElementName: "ЗелёнаяГалка", Path: "_resources/ЗелёнаяГалка/Picture.png", OriginalName: "Picture.png"},
		{ElementName: "ТЗНоменклатурыПоле1", Path: "_resources/ТЗНоменклатурыПоле1/ValuesPicture.png", OriginalName: "ValuesPicture.png"},
	}
	AttachResourcesToForm(form, resources)

	walked := map[string]*IRElement{}
	for _, c := range form.Elements[0].Children {
		walked[c.Name] = c
	}
	if walked["ЗелёнаяГалка"].Picture != "_resources/ЗелёнаяГалка/Picture.png" {
		t.Errorf("ЗелёнаяГалка.Picture = %q", walked["ЗелёнаяГалка"].Picture)
	}
	if walked["ТЗНоменклатурыПоле1"].Values != "_resources/ТЗНоменклатурыПоле1/ValuesPicture.png" {
		t.Errorf("ТЗ.Values = %q", walked["ТЗНоменклатурыПоле1"].Values)
	}
	if walked["БезРесурса"].Picture != "" {
		t.Errorf("БезРесурса.Picture неожиданно установился: %q", walked["БезРесурса"].Picture)
	}
}
