package launcher

import (
	"path/filepath"
	"runtime"
	"testing"
)

// План 47: safeBackupPath защищает от path traversal в {file} URL-параметре.
func TestSafeBackupPath(t *testing.T) {
	dir := t.TempDir()

	good := []string{"backup.sql.gz", "base_2026-06-06_10-00.obz", "db.sqlite"}
	for _, f := range good {
		fp, err := safeBackupPath(dir, f)
		if err != nil {
			t.Errorf("ожидался валидный путь для %q, got err: %v", f, err)
			continue
		}
		want := filepath.Join(dir, f)
		if absWant, _ := filepath.Abs(want); fp != absWant {
			t.Errorf("для %q: got %q, want %q", f, fp, absWant)
		}
	}

	bad := []string{
		"",
		"..",
		"../secret",
		"../../etc/passwd",
		"sub/dir/file",
		"a\x00b",
	}
	if runtime.GOOS == "windows" {
		bad = append(bad, `..\..\windows\system32`, `C:\Windows\system.ini`)
	}
	for _, f := range bad {
		if _, err := safeBackupPath(dir, f); err == nil {
			t.Errorf("ожидалась ошибка для опасного имени %q, но путь принят", f)
		}
	}
}
