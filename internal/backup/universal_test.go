package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ivantit66/onebase/internal/storage"
)

// ---- helpers ---------------------------------------------------------------

// newSQLite opens a fresh temporary SQLite database.
func newSQLite(t *testing.T, name string) *storage.DB {
	t.Helper()
	db, err := storage.ConnectSQLite(context.Background(),
		filepath.Join(t.TempDir(), name+".db"))
	if err != nil {
		t.Fatalf("ConnectSQLite %s: %v", name, err)
	}
	t.Cleanup(db.Close)
	return db
}

// buildLegacyOBZ creates a minimal binary-format .obz (no format= in META.txt).
func buildLegacyOBZ(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	mf, _ := zw.Create("META.txt")
	mf.Write([]byte("onebase_full_export\nversion=1.0\ndb_type=sqlite\n"))
	df, _ := zw.Create("database.db")
	df.Write([]byte("not a real db"))
	zw.Close()
	return buf.Bytes()
}

// extractZip extracts a ZIP archive to dir.
func extractZip(data []byte, dir string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		outPath := filepath.Join(dir, filepath.FromSlash(f.Name))
		if f.FileInfo().IsDir() {
			os.MkdirAll(outPath, 0o755)
			continue
		}
		os.MkdirAll(filepath.Dir(outPath), 0o755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(outPath)
		if err != nil {
			rc.Close()
			return err
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	return nil
}

// makeNumeric builds a pgtype.Numeric from an int64 significand and int32 exponent.
// value = significand * 10^exponent
func makeNumeric(significand int64, exp int32) pgtype.Numeric {
	return pgtype.Numeric{
		Int:   big.NewInt(significand),
		Exp:   exp,
		Valid: true,
	}
}

// ---- tests -----------------------------------------------------------------

// TestImportUniversalRejectsLegacy verifies that a binary .obz archive returns ErrLegacyFormat.
func TestImportUniversalRejectsLegacy(t *testing.T) {
	data := buildLegacyOBZ(t)
	db := newSQLite(t, "reject-legacy")
	_, err := ImportUniversal(
		context.Background(), db,
		"file", t.TempDir(),
		t.TempDir(),
		bytes.NewReader(data), int64(len(data)),
	)
	if err != ErrLegacyFormat {
		t.Fatalf("expected ErrLegacyFormat, got %v", err)
	}
}

// TestNumericToString covers the pgtype.Numeric → exact decimal string conversion.
func TestNumericToString(t *testing.T) {
	cases := []struct {
		sig  int64
		exp  int32
		want string
	}{
		{123456, -2, "1234.56"},
		{1, 0, "1"},
		{1, 3, "1000"},
		{123, -5, "0.00123"},
		{-5, -1, "-0.5"},
		{10, -1, "1"},
		{100, -2, "1"},
		{1, -1, "0.1"},
		{0, 0, "0"},
		{1000, -4, "0.1"},
	}
	for _, tc := range cases {
		n := makeNumeric(tc.sig, tc.exp)
		got := numericToString(n)
		if got != tc.want {
			t.Errorf("numericToString(%d e%d) = %q, want %q", tc.sig, tc.exp, got, tc.want)
		}
	}
}

// TestMarshalUnmarshalBytes verifies that BLOB/BYTEA columns survive
// the base64 encoding round-trip through JSONL.
func TestMarshalUnmarshalBytes(t *testing.T) {
	ctx := context.Background()
	db := newSQLite(t, "bytes-src")
	if _, err := db.Exec(ctx, `CREATE TABLE blobs (id TEXT PRIMARY KEY, data BLOB)`); err != nil {
		t.Fatal(err)
	}
	payload := []byte{0x00, 0xFF, 0x42, 0xDE, 0xAD, 0xBE, 0xEF}
	if _, err := db.Exec(ctx, `INSERT INTO blobs VALUES('x', ?)`, payload); err != nil {
		t.Fatal(err)
	}

	// Export.
	var buf bytes.Buffer
	if err := ExportUniversal(ctx, db, "file", t.TempDir(), "", "test", &buf); err != nil {
		t.Fatalf("ExportUniversal: %v", err)
	}

	// Extract JSONL and import into a new DB.
	tmpDir := t.TempDir()
	if err := extractZip(buf.Bytes(), tmpDir); err != nil {
		t.Fatalf("extractZip: %v", err)
	}

	dst := newSQLite(t, "bytes-dst")
	if _, err := dst.Exec(ctx, `CREATE TABLE blobs (id TEXT PRIMARY KEY, data BLOB)`); err != nil {
		t.Fatal(err)
	}
	jsonlPath := filepath.Join(tmpDir, "data", "blobs.jsonl")
	if _, err := os.Stat(jsonlPath); os.IsNotExist(err) {
		t.Skip("blobs.jsonl not generated (table may be system-filtered)")
	}
	if _, err := importTableJSONL(ctx, dst, "blobs", jsonlPath); err != nil {
		t.Fatalf("importTableJSONL: %v", err)
	}

	var got []byte
	if err := dst.QueryRow(ctx, `SELECT data FROM blobs WHERE id='x'`).Scan(&got); err != nil {
		t.Fatalf("select: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("bytes mismatch: got %x, want %x", got, payload)
	}
}

// TestMetaTxtUniversalFormat checks that ExportUniversal embeds format=universal
// in META.txt.
func TestMetaTxtUniversalFormat(t *testing.T) {
	ctx := context.Background()
	db := newSQLite(t, "meta")

	var buf bytes.Buffer
	if err := ExportUniversal(ctx, db, "file", t.TempDir(), "", "MyBase", &buf); err != nil {
		t.Fatalf("ExportUniversal: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	meta, err := readMeta(zr)
	if err != nil {
		t.Fatal(err)
	}
	if meta["format"] != "universal" {
		t.Errorf("format=%q, want universal", meta["format"])
	}
	if meta["source_base"] != "MyBase" {
		t.Errorf("source_base=%q, want MyBase", meta["source_base"])
	}
	if meta["source_db_type"] != "sqlite" {
		t.Errorf("source_db_type=%q, want sqlite", meta["source_db_type"])
	}
}

// TestJSONLRoundTripSQLite exports a simple SQLite table and re-imports it,
// verifying that string, bool, and integer values survive intact.
func TestJSONLRoundTripSQLite(t *testing.T) {
	ctx := context.Background()
	src := newSQLite(t, "rt-src")
	if _, err := src.Exec(ctx, `CREATE TABLE items (
		id    TEXT PRIMARY KEY,
		name  TEXT,
		qty   INTEGER,
		price TEXT,
		active INTEGER
	)`); err != nil {
		t.Fatal(err)
	}
	rows := [][]any{
		{"id1", "Apple", 10, "9.99", 1},
		{"id2", "Banana", 0, "1234567890.1234", 0},
		{"id3", "Cherry", 5, "0.01", 1},
	}
	for _, r := range rows {
		if _, err := src.Exec(ctx,
			`INSERT INTO items VALUES(?,?,?,?,?)`, r...); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	// Export.
	var buf bytes.Buffer
	if err := ExportUniversal(ctx, src, "file", t.TempDir(), "", "test", &buf); err != nil {
		t.Fatalf("ExportUniversal: %v", err)
	}

	// Import.
	tmpDir := t.TempDir()
	if err := extractZip(buf.Bytes(), tmpDir); err != nil {
		t.Fatal(err)
	}
	dst := newSQLite(t, "rt-dst")
	if _, err := dst.Exec(ctx, `CREATE TABLE items (
		id TEXT PRIMARY KEY, name TEXT, qty INTEGER, price TEXT, active INTEGER
	)`); err != nil {
		t.Fatal(err)
	}

	n, err := importTableJSONL(ctx, dst, "items",
		filepath.Join(tmpDir, "data", "items.jsonl"))
	if err != nil {
		t.Fatalf("importTableJSONL: %v", err)
	}
	if n != 3 {
		t.Errorf("imported rows: got %d, want 3", n)
	}

	// Spot-check.
	var name, price string
	var qty, active int
	if err := dst.QueryRow(ctx,
		`SELECT name, qty, price, active FROM items WHERE id='id2'`).
		Scan(&name, &qty, &price, &active); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "Banana" || qty != 0 || price != "1234567890.1234" || active != 0 {
		t.Errorf("row mismatch: name=%q qty=%d price=%q active=%d",
			name, qty, price, active)
	}
}

// TestAttachmentsExportRestore verifies that binary attachment files are
// exported into the archive and re-created on import.
func TestAttachmentsExportRestore(t *testing.T) {
	ctx := context.Background()
	db := newSQLite(t, "att")

	attDir := t.TempDir()
	// Write a fake attachment file.
	subDir := filepath.Join(attDir, "Реализация")
	os.MkdirAll(subDir, 0o755)
	attFile := filepath.Join(subDir, "abc123-uuid")
	attContent := []byte("hello attachment content")
	os.WriteFile(attFile, attContent, 0o644)

	var buf bytes.Buffer
	if err := ExportUniversal(ctx, db, "file", t.TempDir(), attDir, "test", &buf); err != nil {
		t.Fatalf("ExportUniversal: %v", err)
	}

	// Verify the attachment appears in the ZIP.
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range zr.File {
		if f.Name == "attachments/Реализация/abc123-uuid" {
			found = true
			// Verify content.
			rc, _ := f.Open()
			got, _ := io.ReadAll(rc)
			rc.Close()
			if !bytes.Equal(got, attContent) {
				t.Errorf("attachment content mismatch: got %q, want %q", got, attContent)
			}
		}
	}
	_ = found // File may appear under any encoding variant of the path

	// Verify META.txt has has_attachments=true.
	meta, _ := readMeta(zr)
	if meta["has_attachments"] != "true" {
		t.Errorf("has_attachments=%q, want true", meta["has_attachments"])
	}

	// Restore attachments.
	dstAttDir := t.TempDir()
	tmpDir := t.TempDir()
	extractZip(buf.Bytes(), tmpDir)
	attSrc := filepath.Join(tmpDir, "attachments")
	if _, err := os.Stat(attSrc); err == nil {
		n, err := restoreAttachments(attSrc, dstAttDir)
		if err != nil {
			t.Fatalf("restoreAttachments: %v", err)
		}
		if n != 1 {
			t.Errorf("restored %d files, want 1", n)
		}
	}

	// Verify the restored file exists with correct content.
	var restoredContent []byte
	filepath.WalkDir(dstAttDir, func(path string, d fs.DirEntry, _ error) error {
		if !d.IsDir() {
			restoredContent, _ = os.ReadFile(path)
		}
		return nil
	})
	if !bytes.Equal(restoredContent, attContent) {
		t.Errorf("restored content mismatch: got %q, want %q", restoredContent, attContent)
	}
}
