package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ivantit66/onebase/internal/printform"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// printformsCmd — родительская команда для работы с печатными формами.
var printformsCmd = &cobra.Command{
	Use:   "printforms",
	Short: "Печатные формы OneBase (миграция legacy YAML → макет v2)",
	Long: `Подкоманды для работы с печатными формами OneBase.

OneBase v2 описывает печатную форму декларативным макетом (.layout.yaml):
именованные области ячеек + binding к данным документа. Команда migrate
конвертирует устаревший плоский YAML-формат (title/header/table/footer)
в макет v2.`,
}

var printformsMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Конвертировать legacy YAML печатные формы в макет v2 (.layout.yaml)",
	Long: `Для каждого printforms/*.yaml (устаревший формат) выполняет конвертацию в
макет v2 и пишет рядом <имя>.layout.yaml. Старый .yaml по умолчанию удаляется
(сохранить — флаг --keep). Файлы .os и существующие .layout.yaml не трогаются.

Примеры:
  onebase printforms migrate --project examples/trade
  onebase printforms migrate --project examples/accounting --keep`,
	RunE:          runPrintformsMigrate,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	printformsMigrateCmd.Flags().String("project", ".", "путь к каталогу конфигурации")
	printformsMigrateCmd.Flags().Bool("keep", false, "сохранить исходные .yaml (по умолчанию удаляются)")
	printformsCmd.AddCommand(printformsMigrateCmd)
	rootCmd.AddCommand(printformsCmd)
}

func runPrintformsMigrate(cmd *cobra.Command, _ []string) error {
	dir, _ := cmd.Flags().GetString("project")
	keep, _ := cmd.Flags().GetBool("keep")

	converted, err := migrateLegacyPrintForms(dir, keep)
	if err != nil {
		return err
	}
	if len(converted) == 0 {
		fmt.Fprintln(os.Stdout, "Устаревших печатных форм (.yaml) не найдено — конвертировать нечего.")
		return nil
	}
	fmt.Fprintf(os.Stdout, "Конвертировано форм: %d\n", len(converted))
	for _, c := range converted {
		fmt.Fprintf(os.Stdout, "  %s → %s\n", c.From, c.To)
	}
	if keep {
		fmt.Fprintln(os.Stdout, "\nИсходные .yaml сохранены (--keep). ВНИМАНИЕ: и .yaml, и .layout.yaml")
		fmt.Fprintln(os.Stdout, "одной формы одновременно приведут к коллизии — удалите .yaml вручную.")
	}
	return nil
}

// migrateResult описывает одну конвертированную форму (для вывода).
type migrateResult struct {
	From string
	To   string
}

// migrateLegacyPrintForms конвертирует все устаревшие printforms/*.yaml каталога
// projectDir в макеты v2 (.layout.yaml). keep=false удаляет исходные .yaml.
// Возвращает список конвертированных форм. Отсутствие каталога printforms — не
// ошибка (пустой результат). Файлы *.layout.yaml и *.os пропускаются.
func migrateLegacyPrintForms(projectDir string, keep bool) ([]migrateResult, error) {
	pfDir := filepath.Join(projectDir, "printforms")
	entries, err := os.ReadDir(pfDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("printforms migrate: чтение %s: %w", pfDir, err)
	}

	var out []migrateResult
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Только плоские legacy *.yaml: не *.layout.yaml, не *.os.
		if !strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".layout.yaml") {
			continue
		}
		srcPath := filepath.Join(pfDir, name)
		pf, err := printform.LoadFile(srcPath)
		if err != nil {
			return nil, fmt.Errorf("printforms migrate: %s: %w", name, err)
		}
		lt, err := printform.ConvertLegacy(pf)
		if err != nil {
			return nil, fmt.Errorf("printforms migrate: конвертация %s: %w", name, err)
		}
		data, err := yaml.Marshal(lt)
		if err != nil {
			return nil, fmt.Errorf("printforms migrate: сериализация %s: %w", name, err)
		}
		base := strings.TrimSuffix(name, ".yaml")
		dstPath := filepath.Join(pfDir, base+".layout.yaml")
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return nil, fmt.Errorf("printforms migrate: запись %s: %w", dstPath, err)
		}
		if !keep {
			if err := os.Remove(srcPath); err != nil {
				return nil, fmt.Errorf("printforms migrate: удаление %s: %w", srcPath, err)
			}
		}
		out = append(out, migrateResult{From: name, To: base + ".layout.yaml"})
	}
	return out, nil
}
