package domain_test

import (
	"errors"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func TestParseAmount(t *testing.T) {
	cases := []struct {
		in   string
		dec  int
		want int64
		ok   bool
	}{
		{"25", 2, 2500, true}, {"25.5", 2, 2550, true}, {"25.50", 2, 2550, true}, {"0.01", 2, 1, true},
		{"0", 2, 0, true}, {"007", 2, 700, true}, {" 3.00 ", 2, 300, true}, {"5", 0, 5, true},
		{"12.3456", 4, 123456, true},
		{"", 2, 0, false}, {"-1", 2, 0, false}, {"+1", 2, 0, false}, {"1.234", 2, 0, false},
		{"1.", 2, 0, false}, {".5", 2, 0, false}, {"1e3", 2, 0, false}, {"1,5", 2, 0, false},
		{"abc", 2, 0, false}, {"1.5", 0, 0, false}, {"92233720368547758.07", 2, math.MaxInt64, true},
		{"92233720368547758.08", 2, 0, false}, {"99999999999999999999", 2, 0, false},
	}
	for _, c := range cases {
		got, err := domain.ParseAmount(c.in, c.dec)
		if (err == nil) != c.ok || (c.ok && got != c.want) {
			t.Errorf("ParseAmount(%q,%d) = %d, %v; want %d ok=%v", c.in, c.dec, got, err, c.want, c.ok)
		}
		if err != nil && !errors.Is(err, domain.ErrValidation) {
			t.Errorf("ParseAmount(%q) error code = %v, want validation", c.in, domain.CodeOf(err))
		}
	}
	if _, err := domain.ParseAmount("1", 9); err == nil {
		t.Error("decimals above platform max must be refused")
	}
}

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		v    int64
		dec  int
		want string
	}{
		{2500, 2, "25.00"}, {-2500, 2, "-25.00"}, {1, 2, "0.01"}, {-5, 2, "-0.05"}, {0, 2, "0.00"},
		{7, 0, "7"}, {-7, 0, "-7"}, {123456, 4, "12.3456"}, {5, 4, "0.0005"},
		{math.MaxInt64, 2, "92233720368547758.07"}, {math.MinInt64, 2, "-92233720368547758.08"},
	}
	for _, c := range cases {
		if got := domain.FormatAmount(c.v, c.dec); got != c.want {
			t.Errorf("FormatAmount(%d,%d) = %q want %q", c.v, c.dec, got, c.want)
		}
	}
}

// DOM-005: decimal round trips never change a value, for every supported decimals setting.
func TestAmountRoundTripProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		dec := rng.Intn(domain.MaxDecimals + 1)
		v := rng.Int63()
		if rng.Intn(3) == 0 {
			v = rng.Int63n(1_000_000)
		}
		s := domain.FormatAmount(v, dec)
		got, err := domain.ParseAmount(s, dec)
		if err != nil || got != v {
			t.Fatalf("round trip v=%d dec=%d s=%q got=%d err=%v", v, dec, s, got, err)
		}
	}
}

func TestAddChecked(t *testing.T) {
	if _, ok := domain.AddChecked(math.MaxInt64, 1); ok {
		t.Error("overflow not detected")
	}
	if _, ok := domain.AddChecked(math.MinInt64, -1); ok {
		t.Error("underflow not detected")
	}
	if v, ok := domain.AddChecked(-5, 3); !ok || v != -2 {
		t.Errorf("got %d %v", v, ok)
	}
}

// DOM-005 QA gate: no floating-point type anywhere in money-handling packages.
func TestNoFloatsInMoneyPackages(t *testing.T) {
	re := regexp.MustCompile(`\bfloat(32|64)\b|strconv\.(ParseFloat|FormatFloat)|math/big`)
	for _, dir := range []string{".", "../store", "../service"} {
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if loc := re.Find(b); loc != nil {
				t.Errorf("%s uses %q; money must be integer minor units", f, loc)
			}
		}
	}
}

func TestMigrationsAvoidFloatColumns(t *testing.T) {
	files, _ := filepath.Glob("../store/migrations/*.sql")
	if len(files) == 0 {
		t.Fatal("no migrations found")
	}
	re := regexp.MustCompile(`(?i)\b(REAL|FLOAT|DOUBLE|NUMERIC|DECIMAL)\b`)
	for _, f := range files {
		b, _ := os.ReadFile(f)
		if loc := re.Find(b); loc != nil {
			t.Errorf("%s declares %q column type", f, loc)
		}
	}
}

var testNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
