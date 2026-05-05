package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Account represents a single entry in a chart of accounts.
type Account struct {
	Code   string `yaml:"code"`
	Name   string `yaml:"name"`
	Kind   string `yaml:"kind"`   // active | passive | active_passive
	Parent string `yaml:"parent"` // parent code for hierarchy
}

// ChartOfAccounts is a named set of accounts loaded from YAML.
type ChartOfAccounts struct {
	Name     string    `yaml:"name"`
	Title    string    `yaml:"title"`
	Accounts []Account `yaml:"accounts"`
}

func LoadChartOfAccountsFile(path string) (*ChartOfAccounts, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("accounts: read %s: %w", path, err)
	}
	var chart ChartOfAccounts
	if err := yaml.Unmarshal(data, &chart); err != nil {
		return nil, fmt.Errorf("accounts: parse %s: %w", path, err)
	}
	if chart.Title == "" {
		chart.Title = chart.Name
	}
	for i := range chart.Accounts {
		if chart.Accounts[i].Kind == "" {
			chart.Accounts[i].Kind = "active_passive"
		}
	}
	return &chart, nil
}

func LoadChartOfAccountsDir(dir string) ([]*ChartOfAccounts, error) {
	items, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("accounts: readdir %s: %w", dir, err)
	}
	var charts []*ChartOfAccounts
	for _, item := range items {
		if item.IsDir() || !strings.HasSuffix(item.Name(), ".yaml") {
			continue
		}
		chart, err := LoadChartOfAccountsFile(filepath.Join(dir, item.Name()))
		if err != nil {
			return nil, err
		}
		charts = append(charts, chart)
	}
	return charts, nil
}
