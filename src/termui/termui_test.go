package termui

import (
	"bytes"
	"strings"
	"testing"
)

func TestNeverHasNoANSI(t *testing.T) {
	var out bytes.Buffer
	u := New(&out, Never)
	u.Success("proxy", "127.0.0.1:1080")
	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("Never mode emitted ANSI: %q", out.String())
	}
}

func TestAlwaysHasANSIAndStatusWord(t *testing.T) {
	var out bytes.Buffer
	u := New(&out, Always)
	u.Warning("proxy", "public")
	if !strings.Contains(out.String(), "\x1b[") || !strings.Contains(out.String(), "WARN") {
		t.Fatalf("Always mode output = %q", out.String())
	}
}

func TestParseMode(t *testing.T) {
	for _, mode := range []string{"auto", "always", "never", "AUTO"} {
		if _, err := ParseMode(mode); err != nil {
			t.Fatalf("ParseMode(%q): %v", mode, err)
		}
	}
	if _, err := ParseMode("sometimes"); err == nil {
		t.Fatal("invalid mode accepted")
	}
}
