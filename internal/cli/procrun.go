package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ivantit66/onebase/internal/launcher"
	"github.com/ivantit66/onebase/internal/project"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/ivantit66/onebase/internal/ui"
	"github.com/spf13/cobra"
)

var procrunCmd = &cobra.Command{
	Use:   "procrun",
	Short: "Запустить обработку (Выполнить) офлайн — для отладки",
	Long: `Запускает процедуру Выполнить() обработки вне HTTP-сервера и печатает
вывод Сообщить(). Файловые параметры читаются с диска (автоопределение
кодировки UTF-8/Windows-1251), как при загрузке через браузер.

Пример:
  onebase procrun --id <baseID> --proc ЗагрузкаВыписки \
    --set Действие=Предпросмотр --set Формат=Авто \
    --file ТекстВыгрузки=C:\path\kl_to_1c.txt`,
	RunE: runProcrun,
}

func init() {
	procrunCmd.Flags().String("id", "", "ID базы из реестра ibases")
	procrunCmd.Flags().String("project", ".", "путь к каталогу конфигурации")
	procrunCmd.Flags().String("sqlite", "", "путь к файлу SQLite (альтернатива --db)")
	procrunCmd.Flags().String("db", "", "PostgreSQL DSN (или переменная DATABASE_URL)")
	procrunCmd.Flags().String("proc", "", "имя обработки (обязательно)")
	procrunCmd.Flags().StringArray("set", nil, "параметр обработки: ключ=значение (можно несколько)")
	procrunCmd.Flags().StringArray("file", nil, "файловый параметр: ключ=путь (можно несколько)")
	rootCmd.AddCommand(procrunCmd)
}

func runProcrun(cmd *cobra.Command, _ []string) error {
	procName, _ := cmd.Flags().GetString("proc")
	if procName == "" {
		return fmt.Errorf("укажите --proc <имя обработки>")
	}

	var dir, dsn, sqlitePath, dbType string
	if baseID, _ := cmd.Flags().GetString("id"); baseID != "" {
		store, err := launcher.NewStore()
		if err != nil {
			return fmt.Errorf("ibases store: %w", err)
		}
		base, err := store.Get(baseID)
		if err != nil {
			return fmt.Errorf("база не найдена: %w", err)
		}
		dir, dsn, dbType, sqlitePath = base.Path, base.DB, base.DBType, base.DBPath
	} else {
		dir, _ = cmd.Flags().GetString("project")
		dsn = dsnFromFlags(cmd)
		sqlitePath, _ = cmd.Flags().GetString("sqlite")
		if sqlitePath != "" {
			dbType = "sqlite"
		}
	}

	ctx := context.Background()
	var (
		db  *storage.DB
		err error
	)
	if dbType == "sqlite" {
		if sqlitePath == "" {
			return fmt.Errorf("для SQLite укажите путь к файлу базы")
		}
		db, err = storage.ConnectSQLite(ctx, sqlitePath)
	} else {
		db, err = storage.Connect(ctx, dsn)
	}
	if err != nil {
		return err
	}
	defer db.Close()

	proj, err := project.Load(dir)
	if err != nil {
		return fmt.Errorf("load project: %w", err)
	}
	defer proj.Close()

	strParams := parseKeyVals(cmd, "set")
	fileParams := parseKeyVals(cmd, "file")

	messages, runErr, err := ui.RunProcessorOffline(ctx, proj, db, procName, strParams, fileParams)
	if err != nil {
		return err
	}
	for _, m := range messages {
		fmt.Fprintln(os.Stdout, m)
	}
	if runErr != nil {
		return fmt.Errorf("ошибка выполнения: %w", runErr)
	}
	return nil
}

// parseKeyVals разбирает повторяющиеся флаги вида ключ=значение в map.
func parseKeyVals(cmd *cobra.Command, flag string) map[string]string {
	pairs, _ := cmd.Flags().GetStringArray(flag)
	out := make(map[string]string, len(pairs))
	for _, p := range pairs {
		if i := strings.IndexByte(p, '='); i >= 0 {
			out[p[:i]] = p[i+1:]
		}
	}
	return out
}
