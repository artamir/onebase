package interpreter

import "strings"

// newErrorInfo строит объект ИнформацияОбОшибке как *Struct с полями
// Описание / НомерСтроки / Источник. Возврат через *Struct позволяет
// обращаться к полям (Инфо.Описание) и через Инфо.Свойство("Описание")
// без отдельной ветки в диспетчере MemberExpr.
//
// НомерСтроки/Источник могут быть нулевыми, если ошибка поднята из метода
// объекта (RaiseUserError) без привязки к строке модуля.
func newErrorInfo(ue *userError) *Struct {
	s := &Struct{vals: map[string]any{}}
	set := func(key string, v any) {
		s.keys = append(s.keys, key)
		s.vals[key] = v
	}
	set("описание", ue.Msg)
	set("номерстроки", float64(ue.Line))
	set("источник", baseModuleName(ue.File))
	return s
}

// baseModuleName возвращает имя файла модуля без каталога (Источник в 1С —
// модуль, где возникла ошибка). Поддерживает оба разделителя пути.
func baseModuleName(path string) string {
	if path == "" {
		return ""
	}
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}
	return path
}
