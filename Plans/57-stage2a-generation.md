# План 57 · Этап 2a — Генерация каркаса (бэкенд, метаданные)

**Дата дизайна:** 2026-06-15
**Ветка:** `feature/57-stage2a-generation`
**Статус:** дизайн утверждён, ожидает плана реализации
**Родитель:** [57-ai-configurator-codegen.md](57-ai-configurator-codegen.md) — этап 2 (часть a).
**Опирается на:** этап 0 (песочница, PR #70 — фундамент под будущую генерацию `.os`), этап 1 (`aicontext`, PR #71 — срез конфигурации в промпт).

---

## Контекст и цель

Головная фича плана 57: «опиши задачу по-русски → получи рабочий каркас конфигурации».
Этап 2a — **бэкенд генерации, метаданные-только, без UI и без применения**:
модель через tool-use создаёт объекты в **staging-черновике**, сама прогоняет
`check`, исправляется, а сервер возвращает **diff** предложенных объектов. Применение
и UI — этап 2b.

**Объём (решение пользователя):** только YAML-метаданные (справочники, документы,
регистры накопления/сведений, перечисления, планы счетов, регистры бухгалтерии).
**Без `.os`-модулей** (проводки/обработчики) — тогда авто-`check` не исполняет
сгенерированный код. Генерация модулей (с прогоном через песочницу этапа 0) — позже.

## Целевая архитектура

Самодостаточная сессия генерации в новом файле `internal/launcher/ai_generate.go`,
полностью тестируемая **без LLM** (логику инструментов гоняем напрямую):

```go
// GenChange — один предложенный объект в diff.
type GenChange struct {
	Path       string `json:"path"`        // относительный путь, напр. "catalogs/клиент.yaml"
	Kind       string `json:"kind"`        // "новый" | "изменён"
	NewContent string `json:"newContent"`
	OldContent string `json:"oldContent,omitempty"`
}

// genSession — staging-оверлей конфигурации + накопленные изменения одной генерации.
type genSession struct {
	srcDir  string          // исходная конфигурация базы (materializeProject)
	overlay string          // temp-копия srcDir, куда пишет модель
	changed map[string]bool // относительные пути созданных/изменённых файлов
}

func newGenSession(srcDir string) (*genSession, error) // рекурсивная копия srcDir → temp overlay
func (g *genSession) close()                            // удалить overlay
func (g *genSession) createObject(kind, name, yamlText string) error // тип→подкаталог, запись в overlay
func (g *genSession) check() string                    // CheckDir(overlay)+project.Load → текст ошибок («нет ошибок» / список)
func (g *genSession) showObject(name string) string    // YAML существующего объекта (из overlay) для контекста
func (g *genSession) diff() []GenChange                 // по changed: новый/изменён, old/new content
func (g *genSession) tools() ([]llm.Tool, llm.ToolExecutor) // 3 инструмента для RunWithTools
```

### Маппинг типа объекта → подкаталог (`createObject`)
Соответствует `configcheck.CheckDir` (check.go:73-85). Регистронезависимо, по
синонимам:
- справочник / каталог → `catalogs`
- документ → `documents`
- регистр накопления / регистрнакопления → `registers`
- регистр сведений → `inforegs`
- перечисление → `enums`
- план счетов → `accounts`
- регистр бухгалтерии → `accountregs`

Неизвестный тип → ошибка (модель увидит её в ToolResult и исправится). Имя файла —
`strings.ToLower(name)` + `.yaml`; имя валидируется (без слешей/`..`/пустое — чтобы
модель не вышла за overlay; путь через `filepath.Join`+проверка префикса overlay).

### `check()` — безопасно, без исполнения
`configcheck.CheckDir(overlay)` (чистый парс YAML, без кода) + `project.Load(overlay)`
(ловит кросс-ссылки: «нет такой сущности» и т.п.; модули парсятся, **не** исполняются).
**`CheckQueries` НЕ вызываем** — он компилирует и исполняет запросы; на этапе
метаданных-только это лишнее и означало бы прогон существующего кода базы. Результат —
человекочитаемый текст ошибок (или «Ошибок нет»), который возвращается модели.

### `tools()` — три инструмента (`llm.Tool` + `ToolExecutor`)
- `создать_объект{тип, имя, yaml}` → `createObject`; результат «создан …» / текст ошибки.
- `проверить_конфигурацию{}` → `check`.
- `показать_объект{имя}` → `showObject`.

`ToolExecutor` диспетчеризует по `call.Name`, читает `call.Input["тип"|"имя"|"yaml"]`,
возвращает `llm.ToolResult{ID, Content, IsError}`.

### Хендлер `cfgAIGenerate` (тонкий)
Эндпоинт `POST /bases/{id}/configurator/ai-generate` — в той же auth-группе
конфигуратора, что `cfgAIAssist` (server.go:186). Поток:
1. `materializeProject` → srcDir; `newGenSession(srcDir)`; `defer g.close()`.
2. Системный промпт = роль генератора + **срез конфигурации** `h.configSchemaText`
   (этап 1) + builtin-список + инструкция: создавай объекты YAML, после каждого
   набора вызывай `проверить_конфигурацию`, исправляй ошибки, не выдумывай типы полей.
3. `llm.New(cfg).RunWithTools(ctx, "конфигуратор", req, g.tools())`
   (под 90с-таймаутом, как `cfgAIAssist`). tool-use работает на Anthropic-моделях;
   для прочих провайдеров `RunWithTools` деградирует до обычного ответа (follow-up
   «tool-use для OpenAI/Gemini» — вне этапа).
4. Ответ: `{ok, text: resp.Text, model, changes: g.diff()}`. **Применения нет.**
5. Журнал ИИ (`LogAIQuery`) — как в чате (опционально; токены/модель).

## Тесты (`internal/launcher/ai_generate_test.go`)

Гоняем `genSession` напрямую (без LLM), на временной конфигурации (`t.TempDir()` с
парой объектов):
- `createObject("справочник","Клиент", <валидный YAML>)` → файл появился в
  `overlay/catalogs/клиент.yaml`; `diff()` содержит его как «новый».
- `createObject("документ", …)` с битым YAML → `check()` возвращает текст с ошибкой
  по этому файлу.
- неизвестный тип → `createObject` возвращает ошибку; путь с `..`/слешем → ошибка
  (не пишет вне overlay).
- `showObject("СуществующийСправочник")` → возвращает его YAML.
- `tools()` отдаёт 3 инструмента; `ToolExecutor` на `ToolCall{Name:"создать_объект",
  Input:{тип,имя,yaml}}` пишет объект и возвращает не-IsError; на
  `ToolCall{Name:"проверить_конфигурацию"}` возвращает текст проверки.
- исходная конфигурация базы **не меняется** (пишем только в overlay).

## Verification

1. `go test ./internal/launcher/ -run TestGen -count=1` — зелёные.
2. `go test ./... && go vet ./...` — без регрессий.
3. Ручная (нужна Anthropic-модель в настройках базы): в конфигураторе POST на
   `ai-generate` с «нужен справочник Клиенты и документ Заявка с товарами» → ответ
   содержит `changes` с валидными YAML, `check` в логе прошёл; рабочая конфигурация
   не изменилась.

## Безопасность

- Метаданные-только → авто-`check` не исполняет сгенерированный код.
- Запись только в overlay; имя объекта валидируется (нет выхода за overlay).
- Эндпоинт — в admin-зоне конфигуратора (как `cfgAIAssist`).
- Песочница этапа 0 (`RunSandboxed`) подключается, когда генерация дойдёт до `.os`
  (следующий шаг после 2b) — здесь не нужна.

## YAGNI / вне 2a

- **UI (панель генерации, показ diff, «Применить/Отклонить»)** — этап 2b.
- **Применение diff в рабочую конфигурацию** (файлы/configdb) — этап 2b.
- **Генерация `.os`-модулей** (проводки/обработчики) — отдельный шаг (через песочницу).
- tool-use для OpenAI/Gemini — follow-up плана 51.

## Эстимейт

2.5–3.5 дня: `genSession` (overlay-копия, createObject, маппинг, валидация имени) +
`check`/`showObject`/`diff` + `tools` + хендлер/маршрут + тесты.
