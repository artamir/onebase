# Этап 28 — Автоматический бэкап по расписанию

**Статус:** ⬜ Не начато

## Контекст

Этап 25 добавил `onebase backup` для ручного создания бэкапа. Но в реальном использовании нужен автобэкап — «поставил и забыл». Этот этап интегрирует бэкап с уже готовым планировщиком регламентных заданий (этап 17) и добавляет ротацию старых файлов.

## Синтаксис / UX

### Настройка в `config/app.yaml`

```yaml
backup:
  enabled: true
  schedule: "0 2 * * *"     # каждую ночь в 02:00 (cron-формат)
  keep_last: 7               # хранить последние N бэкапов
  directory: ""              # пусто = ~/.onebase/backups/<base-id>/
```

### UI (Администрирование → Бэкапы)

- Таблица: имя файла, размер, дата создания, кнопка «Скачать» / «Удалить»
- Кнопка **«Создать сейчас»** — запуск внеочередного бэкапа
- Статус последнего автобэкапа (успех/ошибка/время)
- Следующий запланированный бэкап (рассчитывается из `schedule`)

## Хранилище

Файлы: `~/.onebase/backups/<base-id>/backup_<dbname>_<timestamp>.sql.gz`

Метаданные бэкапов в таблице `_backups`:

```sql
CREATE TABLE IF NOT EXISTS _backups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename    TEXT NOT NULL,
    filepath    TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    status      TEXT NOT NULL DEFAULT 'ok',  -- ok | error
    error       TEXT
);
```

## Изменения в коде

**`internal/project/loader.go`**:
- Добавить `BackupConfig` в `AppConfig`:
  ```go
  type BackupConfig struct {
      Enabled   bool   `yaml:"enabled"`
      Schedule  string `yaml:"schedule"`
      KeepLast  int    `yaml:"keep_last"`
      Directory string `yaml:"directory"`
  }
  ```

**`internal/backup/scheduler.go`** (новый файл):
- `RegisterAutoBackup(cfg BackupConfig, connStr string, sched *scheduler.Scheduler)`
- При старте сервера: если `cfg.Enabled` и `cfg.Schedule != ""` → регистрирует cron-задание
- После каждого успешного бэкапа — ротация: удаляет файлы сверх `KeepLast`

**`internal/cli/run.go`** и **`dev.go`**:
- После старта планировщика вызвать `backup.RegisterAutoBackup()`

**`internal/ui/server.go`**:
- Маунт `/ui/admin/backups` → хэндлер списка бэкапов
- `POST /ui/admin/backups/create` → немедленный бэкап

**`internal/ui/handlers.go`**:
- `backupsList` — список из `_backups`
- `backupCreate` — вызов `backup.Dump()` + запись в `_backups`
- `backupDownload` — `http.ServeFile` по `filepath`
- `backupDelete` — удаление файла + записи из `_backups`

## Ротация бэкапов

```go
func rotate(dir string, keepLast int) error {
    files, _ := filepath.Glob(filepath.Join(dir, "backup_*.sql.gz"))
    // sort by mtime desc
    sort.Slice(files, ...)
    for _, f := range files[keepLast:] {
        os.Remove(f)
    }
    return nil
}
```

## Тесты

- `RegisterAutoBackup` с расписанием `* * * * *` создаёт файл в течение минуты
- Ротация: 8 файлов при `keep_last: 7` → удаляет старейший

## Эстимейт

3 дня.
