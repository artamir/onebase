package converter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ivantit66/onebase/internal/converter/parser1c"
	"github.com/ivantit66/onebase/internal/converter/writer"
	"gopkg.in/yaml.v3"
)

// Options настройки конвертации.
type Options struct {
	// SourceDir — путь к выгрузке конфигурации 1С (папка с Catalogs/, Documents/, ...)
	SourceDir string
	// OutDir — куда писать результат (создаётся если нет)
	OutDir string
}

// Convert читает 1С XML-выгрузку и создаёт onebase-проект в OutDir.
func Convert(opts Options) (*writer.ConversionReport, error) {
	if opts.SourceDir == "" {
		return nil, fmt.Errorf("convert: source dir is required")
	}
	if opts.OutDir == "" {
		return nil, fmt.Errorf("convert: output dir is required")
	}

	dump, err := parser1c.ParseDir(opts.SourceDir)
	if err != nil {
		return nil, fmt.Errorf("convert: parse: %w", err)
	}

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return nil, err
	}

	report := &writer.ConversionReport{}

	// Справочники
	if err := writer.WriteCatalogs(dump.Catalogs, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write catalogs: %w", err)
	}

	// Документы
	if err := writer.WriteDocuments(dump.Documents, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write documents: %w", err)
	}

	// Регистры накопления
	if err := writer.WriteRegisters(dump.Registers, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write registers: %w", err)
	}

	// Перечисления
	if err := writer.WriteEnums(dump.Enums, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write enums: %w", err)
	}

	// Константы
	if err := writer.WriteConstants(dump.Constants, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write constants: %w", err)
	}

	// Регистры сведений
	if err := writer.WriteInfoRegisters(dump.InfoRegisters, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write info registers: %w", err)
	}

	// Регистры бухгалтерии
	if err := writer.WriteAccountRegisters(dump.AccountRegisters, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write account registers: %w", err)
	}

	// Планы счетов
	if err := writer.WriteChartsOfAccounts(dump.ChartsOfAccounts, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write charts of accounts: %w", err)
	}

	// Регламентные задания
	if err := writer.WriteScheduledJobs(dump.ScheduledJobs, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write scheduled jobs: %w", err)
	}

	// Общие модули
	if err := writer.WriteModules(dump.Modules, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write modules: %w", err)
	}

	// Обработки
	if err := writer.WriteProcessors(dump.Processors, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write processors: %w", err)
	}

	// DSL-заглушки
	if err := writer.WriteDSLStubs(dump.Documents, opts.SourceDir, opts.OutDir, report); err != nil {
		return nil, fmt.Errorf("convert: write dsl stubs: %w", err)
	}

	// app.yaml — use source dir name (the 1C config folder), not the temp workspace dir
	appName := filepath.Base(opts.SourceDir)
	if appName == "" || appName == "." {
		appName = filepath.Base(opts.OutDir)
	}
	if err := writeAppYAML(opts.OutDir, appName); err != nil {
		return nil, err
	}

	// Пропущенные объекты
	for _, s := range dump.SkippedDirs {
		report.Skipped = append(report.Skipped, s.Kind+"/"+s.Name)
	}

	// Записать отчёт в файл
	reportPath := filepath.Join(opts.OutDir, "conversion_report.txt")
	os.WriteFile(reportPath, []byte(report.String()), 0o644)

	return report, nil
}

type appConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

func writeAppYAML(outDir, name string) error {
	configDir := filepath.Join(outDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	cfg := appConfig{Name: name, Version: "1.0"}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "app.yaml"), data, 0o644)
}
