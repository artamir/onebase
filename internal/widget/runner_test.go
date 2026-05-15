package widget

import (
	"strings"
	"testing"
)

func TestFormatKPI_Money(t *testing.T) {
	got := formatKPI(1234567.5, "money")
	// 1 234 567,50 ₽ — NBSP is replaced with regular space in groupDigits
	want := "1 234 567,50 ₽"
	if got != want {
		t.Errorf("money: got %q, want %q", got, want)
	}
}

func TestFormatKPI_Number(t *testing.T) {
	if got := formatKPI(42.0, "number"); got != "42" {
		t.Errorf("number 42: got %q", got)
	}
	if got := formatKPI(1000000.0, "number"); got != "1 000 000" {
		t.Errorf("number 1m: got %q", got)
	}
}

func TestFormatKPI_Percent(t *testing.T) {
	if got := formatKPI(12.345, "percent"); got != "12.3%" {
		t.Errorf("percent: got %q", got)
	}
}

func TestFormatKPI_DefaultInteger(t *testing.T) {
	if got := formatKPI(5.0, ""); got != "5" {
		t.Errorf("default int: got %q", got)
	}
}

func TestFormatKPI_DefaultFloat(t *testing.T) {
	got := formatKPI(3.14, "")
	if got != "3.14" {
		t.Errorf("default float: got %q", got)
	}
}

func TestGroupDigits(t *testing.T) {
	cases := map[int64]string{
		0:        "0",
		7:        "7",
		1234:     "1 234",
		12345:    "12 345",
		1000000:  "1 000 000",
		12345678: "12 345 678",
	}
	for in, want := range cases {
		if got := groupDigits(in); got != want {
			t.Errorf("groupDigits(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstScalar(t *testing.T) {
	row := map[string]any{"x": 42}
	if got := firstScalar(row); got != 42 {
		t.Errorf("firstScalar = %v", got)
	}
	if got := firstScalar(map[string]any{}); got != nil {
		t.Errorf("empty row should give nil, got %v", got)
	}
}

func TestToFloat(t *testing.T) {
	if toFloat(int64(42)) != 42 {
		t.Error("int64")
	}
	if toFloat("3.14") != 3.14 {
		t.Error("string")
	}
	if toFloat(nil) != 0 {
		t.Error("nil")
	}
}

func TestFormatMoney_Negative(t *testing.T) {
	got := formatMoney(-1500.25)
	if !strings.Contains(got, "-") {
		t.Errorf("expected leading -, got %q", got)
	}
	if !strings.Contains(got, "25") {
		t.Errorf("expected fractional 25 kop, got %q", got)
	}
}
