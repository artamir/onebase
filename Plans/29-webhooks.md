# Этап 29 — Webhook-уведомления

**Статус:** ⬜ Не начато

## Контекст

HTTP-клиент в DSL (этап 19) позволяет отправлять запросы вручную из скриптов. Но для интеграции с внешними системами (Telegram-бот, CRM, мессенджеры, ERP) нужны автоматические webhook-вызовы на события платформы: «документ записан», «документ проведён», «элемент справочника изменён».

## Синтаксис / UX

### Настройка в `config/app.yaml`

```yaml
webhooks:
  - name: "Уведомление в Telegram"
    on: document.post              # событие (см. таблицу ниже)
    filter:
      entity: РеализацияТоваров   # только для этого документа
    url: "https://api.telegram.org/bot<TOKEN>/sendMessage"
    method: POST
    headers:
      Content-Type: application/json
    body: |
      {"chat_id": "-100XXXXXX", "text": "Реализация {{id}} на сумму {{Сумма}} проведена"}
    timeout: 10           # секунд
    retry: 2              # повторов при ошибке
```

### Поддерживаемые события

| Событие | Когда срабатывает |
|---|---|
| `document.save` | После записи документа |
| `document.post` | После проведения документа |
| `document.unpost` | После отмены проведения |
| `catalog.save` | После записи элемента справочника |
| `catalog.delete` | После удаления элемента |
| `report.run` | После выполнения отчёта |

### Шаблон тела запроса

В поле `body` доступны переменные из контекста события:
- `{{id}}` — UUID объекта
- `{{entity}}` — имя объекта
- `{{user}}` — логин пользователя
- `{{timestamp}}` — время события (ISO 8601)
- `{{Поле}}` — любое поле записи

## Изменения в коде

**`internal/project/loader.go`**:
```go
type WebhookConfig struct {
    Name    string            `yaml:"name"`
    On      string            `yaml:"on"`
    Filter  map[string]string `yaml:"filter"`
    URL     string            `yaml:"url"`
    Method  string            `yaml:"method"`
    Headers map[string]string `yaml:"headers"`
    Body    string            `yaml:"body"`
    Timeout int               `yaml:"timeout"`
    Retry   int               `yaml:"retry"`
}
```

**`internal/webhook/` (новый пакет)**:
- `Dispatcher` — хранит список webhook-конфигов, проверяет фильтры, отправляет HTTP-запрос
- `Dispatch(ctx, event string, entityName string, record map[string]any, user string)`
- Шаблонизация тела через `text/template`
- Retry-логика с экспоненциальной задержкой
- Асинхронная отправка (goroutine) — не блокирует основной поток

**`internal/storage/crud.go`**, **`posting.go`**:
- После успешной записи / проведения — вызов `dispatcher.Dispatch(...)`

**Таблица `_webhook_log`**:
```sql
CREATE TABLE IF NOT EXISTS _webhook_log (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_name TEXT NOT NULL,
    event        TEXT NOT NULL,
    entity       TEXT NOT NULL,
    record_id    UUID,
    url          TEXT NOT NULL,
    status_code  INT,
    error        TEXT,
    duration_ms  INT,
    fired_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Администрирование → Webhook-лог**:
- Таблица последних 200 вызовов с фильтром по имени и статусу
- Кнопка «Повторить» (retry последнего failed)

## Тесты

- Мок HTTP-сервера регистрирует входящий запрос при `document.save`
- Retry: при 500 от сервера выполняется ≤ N повторов
- Шаблон `{{Сумма}}` подставляет значение из записи

## Эстимейт

4–5 дней.
