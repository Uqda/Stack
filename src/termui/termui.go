// Package termui provides small, automation-safe terminal styling helpers.
package termui

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Mode string

const (
	Auto   Mode = "auto"
	Always Mode = "always"
	Never  Mode = "never"
)

type UI struct {
	out   io.Writer
	color bool
}

func ParseMode(value string) (Mode, error) {
	switch Mode(strings.ToLower(value)) {
	case Auto, Always, Never:
		return Mode(strings.ToLower(value)), nil
	default:
		return "", fmt.Errorf("invalid color mode %q (expected auto, always, or never)", value)
	}
}

func New(out io.Writer, mode Mode) UI {
	color := mode == Always
	if mode == Auto {
		color = os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
		if f, ok := out.(*os.File); ok {
			if info, err := f.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
				color = false
			}
		} else {
			color = false
		}
	}
	return UI{out: out, color: color}
}

func (u UI) paint(code, value string) string {
	if !u.color {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}

func (u UI) Heading(value string) { fmt.Fprintln(u.out, u.paint("1;36", value)) }
func (u UI) Success(label, value string) {
	fmt.Fprintf(u.out, "%s %-12s %s\n", u.paint("1;32", "PASS"), label, value)
}
func (u UI) Warning(label, value string) {
	fmt.Fprintf(u.out, "%s %-12s %s\n", u.paint("1;33", "WARN"), label, value)
}
