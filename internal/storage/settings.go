package storage

import (
	"context"
	"fmt"
	"strings"
)

// AuditSettings — настройки журнала регистрации (аналог «Журнала регистрации»
// в 1С). Это свойство конкретной информационной базы, а не git-версионируемой
// конфигурации, поэтому хранится в служебной таблице _settings.
type AuditSettings struct {
	Enabled bool // вести журнал регистрации вообще
	Create  bool // регистрировать создание объектов
	Update  bool // регистрировать изменение объектов
	Delete  bool // регистрировать удаление объектов
	Post    bool // регистрировать проведение / отмену проведения
	Login   bool // регистрировать вход / выход пользователей
}

// DefaultAuditSettings — журнал включён, пишутся изменения данных и проведение;
// вход/выход по умолчанию не пишется (шумно для однопользовательских баз).
func DefaultAuditSettings() AuditSettings {
	return AuditSettings{Enabled: true, Create: true, Update: true, Delete: true, Post: true, Login: false}
}

// auditSettingKeys связывает ключи _settings с полями AuditSettings.
func auditSettingKeys(s *AuditSettings) map[string]*bool {
	return map[string]*bool{
		"audit.enabled": &s.Enabled,
		"audit.create":  &s.Create,
		"audit.update":  &s.Update,
		"audit.delete":  &s.Delete,
		"audit.post":    &s.Post,
		"audit.login":   &s.Login,
	}
}

// EnsureSettingsSchema создаёт служебную key-value таблицу _settings.
func (db *DB) EnsureSettingsSchema(ctx context.Context) error {
	if _, err := db.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS _settings (key TEXT PRIMARY KEY, value TEXT NOT NULL DEFAULT '')`,
	); err != nil {
		return fmt.Errorf("settings: create _settings: %w", err)
	}
	return nil
}

// GetAuditSettings читает настройки журнала из _settings. Отсутствующие ключи
// (или отсутствующая таблица) дают значения по умолчанию.
func (db *DB) GetAuditSettings(ctx context.Context) AuditSettings {
	s := DefaultAuditSettings()
	rows, err := db.Query(ctx, `SELECT key, value FROM _settings WHERE key LIKE 'audit.%'`)
	if err != nil {
		return s
	}
	defer rows.Close()
	keys := auditSettingKeys(&s)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		if ptr, ok := keys[k]; ok {
			*ptr = v == "1" || strings.EqualFold(v, "true")
		}
	}
	return s
}

// SaveAuditSettings сохраняет настройки журнала в _settings.
func (db *DB) SaveAuditSettings(ctx context.Context, s AuditSettings) error {
	if err := db.EnsureSettingsSchema(ctx); err != nil {
		return err
	}
	d := db.dialect
	q := fmt.Sprintf(
		`INSERT INTO _settings (key, value) VALUES (%s, %s)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		d.Placeholder(1), d.Placeholder(2))
	for k, ptr := range auditSettingKeys(&s) {
		v := "0"
		if *ptr {
			v = "1"
		}
		if _, err := db.Exec(ctx, q, k, v); err != nil {
			return fmt.Errorf("settings: save %s: %w", k, err)
		}
	}
	return nil
}
