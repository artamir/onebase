package onec_forms

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotImplemented возвращается фасадными функциями до реализации
// соответствующих этапов плана 37 (этап 5 — экспорт, этап 6 — Validate).
var ErrNotImplemented = errors.New("onec_forms: not implemented yet (см. План 37)")

// ImportOptions задаёт пути и метаданные для фасада ImportFromOneC.
type ImportOptions struct {
	// XMLPath — путь к Form.xml (обязателен).
	XMLPath string
	// BSLPath — путь к Module.bsl. Если файл отсутствует, импорт продолжится без модуля.
	BSLPath string
	// ItemsDir — путь к папке Items/ с бинарными ресурсами. Может быть пустым/отсутствовать.
	ItemsDir string

	// EntityName — имя сущности OneBase, к которой привязывается форма.
	EntityName string
	// FormName — имя формы (по умолчанию вытаскивается из имени каталога или ставится "Форма").
	FormName string
	// FormKind — object|list|choice|folder|custom (по умолчанию "custom").
	FormKind string

	// DstYAMLPath — путь к создаваемому .form.yaml.
	DstYAMLPath string
	// DstOSPath — путь к создаваемому .form.os (рядом с YAML).
	DstOSPath string
	// DstResourcesDir — каталог для _resources/.
	DstResourcesDir string
}

// ImportFromOneC читает форму из выгрузки 1С (Form.xml + Module.bsl + Items/*)
// и записывает её в проект OneBase как .form.yaml + .form.os + _resources/.
//
// Возвращает ImportReport с путями созданных файлов и списком предупреждений
// от парсера XML, нормализации, BSL-лексера и копирования ресурсов.
func ImportFromOneC(opts ImportOptions) (*ImportReport, error) {
	if opts.XMLPath == "" {
		return nil, fmt.Errorf("ImportFromOneC: XMLPath обязателен")
	}
	if opts.DstYAMLPath == "" {
		return nil, fmt.Errorf("ImportFromOneC: DstYAMLPath обязателен")
	}

	report := &ImportReport{}

	// 1. Парсим Form.xml.
	form, xmlWarns, err := ReadFormXML(opts.XMLPath)
	if err != nil {
		return nil, fmt.Errorf("read xml: %w", err)
	}
	report.Warnings = append(report.Warnings, xmlWarns...)

	// 2. Нормализация имён 1С → OneBase.
	normWarns := NormalizeForImport(form)
	report.Warnings = append(report.Warnings, normWarns...)

	// 3. Метаданные формы (entity/name/kind заполняются опциями).
	form.Entity = opts.EntityName
	if opts.FormName != "" {
		form.Name = opts.FormName
	} else if form.Name == "" {
		form.Name = "Форма"
	}
	if opts.FormKind != "" {
		form.Kind = opts.FormKind
	} else if form.Kind == "" {
		form.Kind = "custom"
	}

	// 4. Бинарные ресурсы (если есть Items/).
	if opts.ItemsDir != "" && opts.DstResourcesDir != "" {
		resources, resWarns, err := CopyResources(opts.ItemsDir, opts.DstResourcesDir)
		if err != nil {
			return report, fmt.Errorf("copy resources: %w", err)
		}
		report.Warnings = append(report.Warnings, resWarns...)
		AttachResourcesToForm(form, resources)
		if len(resources) > 0 {
			report.ResourcesDir = opts.DstResourcesDir
		}
	}

	// 5. Записываем YAML.
	if err := os.MkdirAll(filepath.Dir(opts.DstYAMLPath), 0o755); err != nil {
		return report, fmt.Errorf("mkdir for yaml: %w", err)
	}
	if err := WriteFormYAML(form, opts.DstYAMLPath); err != nil {
		return report, fmt.Errorf("write yaml: %w", err)
	}
	report.YAMLPath = opts.DstYAMLPath

	// 6. Module.bsl → .form.os.
	if opts.BSLPath != "" && opts.DstOSPath != "" {
		procs, bslWarns, err := ReadBSL(opts.BSLPath)
		if err != nil {
			return report, fmt.Errorf("read bsl: %w", err)
		}
		report.Warnings = append(report.Warnings, bslWarns...)
		if len(procs) > 0 {
			dsl := EmitDSLSource(procs)
			if err := os.MkdirAll(filepath.Dir(opts.DstOSPath), 0o755); err != nil {
				return report, fmt.Errorf("mkdir for os: %w", err)
			}
			if err := WriteFormOS(dsl, opts.DstOSPath); err != nil {
				return report, fmt.Errorf("write os: %w", err)
			}
			report.ModulePath = opts.DstOSPath
		}
	}

	return report, nil
}

// ExportToOneC обратное направление: читает .form.yaml + .form.os
// из проекта OneBase и записывает Form.xml + Module.bsl + Items/*
// в указанный каталог.
//
// Реализуется в этапе 5 плана 37.
func ExportToOneC(yamlPath, osPath, dstFormDir string) (*ExportReport, error) {
	return nil, ErrNotImplemented
}

// Validate проверяет корректность .form.yaml: схема, типы реквизитов,
// существование data_path, наличие процедур-обработчиков в .form.os.
// Возвращает список предупреждений (даже при отсутствии ошибок).
//
// Реализуется в этапе 6 плана 37.
func Validate(yamlPath string) ([]Warning, error) {
	return nil, ErrNotImplemented
}
