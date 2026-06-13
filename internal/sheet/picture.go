package sheet

import (
	"encoding/base64"
	"strings"
)

// Поддержка картинок в ячейках (Cell.Picture) для HTML- и PDF-рендеров
// (план 64, этап 7.1).
//
// Формат Picture. DSL-метод ТабличныйДокумент.Нарисовать(Объект) и свойство
// Ячейка.Рисунок кладут в Cell.Picture ПРОИЗВОЛЬНУЮ строку — отдельного типа
// «картинка» в движке нет. Поэтому рендереры поддерживают реалистичный для
// печатных форм формат:
//
//   - data-URI с base64: "data:image/png;base64,iVBOR..." (PNG/JPEG/GIF).
//     Основной путь: картинку кладут в макет/конфигурацию как самодостаточную
//     строку (без файловой системы и эндпоинтов вложений).
//   - http(s)-URL: "https://example.com/logo.png" — только в HTML (<img src>);
//     в PDF внешние URL не загружаются (нет сети в рендере — осознанно).
//
// Сырой base64 без префикса data: НЕ поддерживается (неоднозначно — тип
// изображения неизвестен). Пути к файлам/вложениям не поддерживаются: вложения
// (#43 richtext) появятся отдельным механизмом и лягут на data-URI.

// pictureKind различает источник картинки в ячейке.
type pictureKind int

const (
	picNone    pictureKind = iota
	picDataURI             // data:image/...;base64,XXXX
	picURL                 // http(s)://...
)

// classifyPicture определяет вид строки-картинки.
func classifyPicture(s string) pictureKind {
	s = strings.TrimSpace(s)
	if s == "" {
		return picNone
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "data:image/") && strings.Contains(low, ";base64,") {
		return picDataURI
	}
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
		return picURL
	}
	return picNone
}

// decodeDataURIImage разбирает data-URI картинки в сырые байты и тип для fpdf
// ("PNG"/"JPG"/"GIF"). ok=false, если это не поддерживаемый data-URI картинки.
func decodeDataURIImage(s string) (data []byte, fpdfType string, ok bool) {
	s = strings.TrimSpace(s)
	if classifyPicture(s) != picDataURI {
		return nil, "", false
	}
	// data:image/<subtype>;base64,<payload>
	comma := strings.IndexByte(s, ',')
	if comma < 0 {
		return nil, "", false
	}
	header := strings.ToLower(s[:comma])
	payload := s[comma+1:]

	var t string
	switch {
	case strings.Contains(header, "image/png"):
		t = "PNG"
	case strings.Contains(header, "image/jpeg"), strings.Contains(header, "image/jpg"):
		t = "JPG"
	case strings.Contains(header, "image/gif"):
		t = "GIF"
	default:
		return nil, "", false
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload))
	if err != nil || len(raw) == 0 {
		return nil, "", false
	}
	return raw, t, true
}
