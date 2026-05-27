// Package entityservice инкапсулирует логику сохранения сущностей (справочников
// и документов): запуск DSL-хука OnWrite/OnPost, упсёрт + табличные части +
// движения + проведение — в одной транзакции.
//
// Зачем выделено: раньше эта логика жила только в internal/ui (методы submit /
// submitEdit на *Server). REST API в internal/api делал упрощённый Upsert без
// хука/ТЧ/движений/проведения — то есть для API программа фактически работала
// только как голый CRUD без бизнес-правил. Теперь обе стороны зовут Service.Save,
// и при необходимости отличаются только тем, *как* они собирают DSL-переменные
// и пред-обработку объекта (см. PrepareHook / BuildVars).
package entityservice

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/metadata"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
)

// SetPeriodFromFields выставляет в mc период по первому date-полю сущности.
// Регистронезависимый поиск ключа: формы кладут PascalCase, Object.Set —
// lowercase. Прежний прямой `fields[f.Name]` промахивался → period оставался
// time.Now() и движения дрейфовали по часовым поясам.
func SetPeriodFromFields(mc *runtime.MovementsCollector, entity *metadata.Entity, fields map[string]any) {
	for _, f := range entity.Fields {
		if f.Type != metadata.FieldTypeDate {
			continue
		}
		low := strings.ToLower(f.Name)
		for k, v := range fields {
			if strings.ToLower(k) != low {
				continue
			}
			if t := runtime.AsTime(v); !t.IsZero() {
				mc.SetPeriod(t)
			}
			break
		}
		return
	}
}

// Service выполняет сохранение объектов вместе с побочными эффектами.
type Service struct {
	Store  *storage.DB
	Reg    *runtime.Registry
	Interp *interpreter.Interpreter

	// PrepareHook вызывается перед запуском DSL-хука. Caller использует это
	// чтобы обогатить obj (например, заменить UUID-строки в полях шапки на
	// *interpreter.Ref{UUID,Name} — нужно чтобы Строка(ref) и ЗначениеРеквизита
	// работали в OnWrite/OnPost так же, как при вызове из обработки).
	// Может быть nil — тогда obj передаётся в хук «как есть».
	PrepareHook func(ctx context.Context, entity *metadata.Entity, obj *runtime.Object)

	// EnrichTPRows обогащает строки табличной части (аналог PrepareHook для ТЧ).
	// Может быть nil.
	EnrichTPRows func(ctx context.Context, tp metadata.TablePart, rows []map[string]any)

	// BuildVars собирает DSL-extraVars для контекста caller'а. mc — обязательный
	// (Движения). msgs (если не nil) — collector для builtin Сообщить, чтобы
	// caller мог отдать сообщения пользователю/в журнал.
	// Может быть nil — DSL-хук запустится без extraVars (тогда Сообщить, HTTP,
	// Справочники и т.п. в нём не будут работать).
	BuildVars func(ctx context.Context, mc *runtime.MovementsCollector, msgs *[]string) map[string]any
}

// SaveRequest — входной DTO для Service.Save.
type SaveRequest struct {
	Entity *metadata.Entity
	ID     uuid.UUID
	IsNew  bool // true → Upsert + авто-сценарии для нового объекта; false → UpsertVersioned

	Fields        map[string]any
	TablePartRows map[string][]map[string]any

	// Action: "" (просто Записать) | "post" | "post_and_close".
	// Для документов с Posting=true и Action=post* запускается OnPost вместо
	// OnWrite и в конце сохранения выставляется posted=true.
	Action string

	// ExpectedVersion — только для !IsNew. nil ⇒ без проверки optimistic
	// lock (поведение совместимо с прежним Upsert). Не-nil ⇒ UpsertVersioned
	// вернёт storage.ErrVersionConflict при несовпадении версии.
	ExpectedVersion *int64
}

// SaveResult — результат Service.Save.
type SaveResult struct {
	ID          uuid.UUID
	DSLError    string                      // если не пусто — хук вернул ошибку, БД не изменена
	DSLMessages []string                    // сообщения из builtin Сообщить
	Movements   *runtime.MovementsCollector // для отладки/инспекции (заполняется хуком OnPost)
}

// Save выполняет полный цикл сохранения: prepare → run hook → tx (upsert +
// table parts + movements + posting).
//
// Возвращает (result, nil) при успехе. Если DSL-хук вернул ошибку — это НЕ
// технический сбой: возвращается result.DSLError != "" и err == nil, caller
// сам решает как показать ошибку. Технические ошибки (БД, network) возвращаются
// как err != nil (включая storage.ErrVersionConflict при !IsNew с конфликтом
// версий — caller должен проверить errors.Is).
func (s *Service) Save(ctx context.Context, req SaveRequest) (SaveResult, error) {
	mc := runtime.NewMovementsCollector(req.Entity.Name, req.ID)
	SetPeriodFromFields(mc, req.Entity, req.Fields)

	obj := &runtime.Object{
		Type:          req.Entity.Name,
		Kind:          req.Entity.Kind,
		ID:            req.ID,
		Fields:        req.Fields,
		TablePartRows: req.TablePartRows,
	}

	// Pre-hook enrichment: даём caller'у заменить UUID-строки на *Ref и т.п.
	if s.PrepareHook != nil {
		s.PrepareHook(ctx, req.Entity, obj)
	}
	if s.EnrichTPRows != nil {
		for _, tp := range req.Entity.TableParts {
			if rows, ok := obj.TablePartRows[tp.Name]; ok {
				s.EnrichTPRows(ctx, tp, rows)
			}
		}
	}

	// Выбор хука: OnPost при проведении документа, иначе OnWrite.
	isPosting := req.Entity.Posting && (req.Action == "post" || req.Action == "post_and_close")
	hookName := "OnWrite"
	if isPosting {
		hookName = "OnPost"
	}
	proc := s.Reg.GetProcedure(req.Entity.Name, hookName)

	var msgs []string
	if proc != nil {
		var vars map[string]any
		if s.BuildVars != nil {
			vars = s.BuildVars(ctx, mc, &msgs)
		}
		if err := s.Interp.Run(proc, obj, vars); err != nil {
			if dslErr, ok := err.(*interpreter.DSLError); ok {
				return SaveResult{ID: req.ID, DSLError: dslErr.Error(), DSLMessages: msgs, Movements: mc}, nil
			}
			return SaveResult{ID: req.ID, DSLError: err.Error(), DSLMessages: msgs, Movements: mc}, nil
		}
	}

	// Транзакция: upsert + ТЧ + движения + проведение.
	err := s.Store.WithTx(ctx, func(ctx context.Context) error {
		if req.IsNew || req.ExpectedVersion == nil {
			if err := s.Store.Upsert(ctx, req.Entity.Name, req.ID, obj.Fields, req.Entity); err != nil {
				return err
			}
		} else {
			if err := s.Store.UpsertVersioned(ctx, req.Entity.Name, req.ID, obj.Fields, req.Entity, req.ExpectedVersion); err != nil {
				return err
			}
		}
		for _, tp := range req.Entity.TableParts {
			rows := req.TablePartRows[tp.Name]
			if rows == nil {
				rows = []map[string]any{}
			}
			if err := s.Store.UpsertTablePartRows(ctx, req.Entity.Name, tp.Name, req.ID, rows, tp); err != nil {
				return err
			}
		}
		if err := s.writeMovements(ctx, req.Entity.Name, req.ID, mc); err != nil {
			return err
		}
		if req.Entity.Posting {
			if isPosting {
				return s.Store.SetPosted(ctx, req.Entity.Name, req.ID, true)
			}
			// «Записать» для уже проведённого документа (только при редактировании,
			// при IsNew проведения нет в принципе) — сбрасываем движения по всем
			// регистрам и снимаем флаг проведения. Это правило сохранено из
			// прежнего ui.submitEdit.
			if !req.IsNew {
				for _, reg := range s.Reg.Registers() {
					if err := s.Store.WriteMovements(ctx, reg.Name, req.Entity.Name, req.ID, nil, reg, nil); err != nil {
						return err
					}
				}
				return s.Store.SetPosted(ctx, req.Entity.Name, req.ID, false)
			}
		}
		return nil
	})
	if err != nil {
		return SaveResult{}, err
	}

	return SaveResult{ID: req.ID, DSLMessages: msgs, Movements: mc}, nil
}

// writeMovements распределяет накопленные в mc движения по нужным типам
// регистров (накопления, счетов, сведений). Вынесено из ui.Server.saveMovements.
func (s *Service) writeMovements(ctx context.Context, docType string, docID uuid.UUID, mc *runtime.MovementsCollector) error {
	for regName, rows := range mc.All() {
		if reg := s.Reg.GetRegister(regName); reg != nil {
			if err := s.Store.WriteMovements(ctx, regName, docType, docID, rows, reg, mc.Period); err != nil {
				return err
			}
			continue
		}
		if ar := s.Reg.GetAccountRegister(regName); ar != nil {
			if err := s.Store.WriteAccountMovements(ctx, regName, docType, docID, rows, ar, mc.Period); err != nil {
				return err
			}
			continue
		}
		if ir := s.Reg.GetInfoRegister(regName); ir != nil {
			if err := s.Store.WriteInfoMovements(ctx, regName, docType, docID, rows, ir, mc.Period); err != nil {
				return err
			}
		}
	}
	return nil
}
