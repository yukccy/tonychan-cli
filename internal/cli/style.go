package cli

import (
	"io"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// Style decides how output looks: colour and clickable links only on a real terminal, and
// never when NO_COLOR is set. Width is the terminal's, or 80 when it can't be known.
type Style struct {
	Color     bool
	TrueColor bool
	Links     bool
	Width     int
}

func detectStyle(w io.Writer) Style {
	s := Style{Width: 80}
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return s
	}
	if width, _, err := term.GetSize(int(f.Fd())); err == nil && width > 0 {
		s.Width = width
	}
	dumb := os.Getenv("TERM") == "dumb"
	s.Color = os.Getenv("NO_COLOR") == "" && !dumb
	s.TrueColor = s.Color && (os.Getenv("COLORTERM") == "truecolor" || os.Getenv("COLORTERM") == "24bit")
	s.Links = !dumb
	return s
}

func (s Style) sgr(code, text string) string {
	if !s.Color || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// Red is the site's accent, #C4211A, falling back to the terminal's own red.
func (s Style) Red(t string) string {
	if s.TrueColor {
		return s.sgr("38;2;196;33;26", t)
	}
	return s.sgr("31", t)
}

func (s Style) Bold(t string) string { return s.sgr("1", t) }
func (s Style) Dim(t string) string  { return s.sgr("2", t) }

// Label is the small upper-case heading used for sections.
func (s Style) Label(t string) string { return s.Bold(strings.ToUpper(t)) }

// Link makes text a clickable OSC 8 hyperlink to url where the terminal supports it.
func (s Style) Link(url, text string) string {
	if !s.Links || url == "" {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// truncate shortens t to fit in width terminal cells, ending with an ellipsis.
func truncate(t string, width int) string {
	if width <= 0 {
		return ""
	}
	return runewidth.Truncate(t, width, "…")
}

// pad right-pads t with spaces to width cells.
func pad(t string, width int) string {
	return runewidth.FillRight(t, width)
}

// field pads styled text to width cells, measuring the plain text it was made from, so escape
// codes don't throw the columns off.
func field(styled, plain string, width int) string {
	return styled + strings.Repeat(" ", max(width-runewidth.StringWidth(plain), 1))
}

// bare drops the scheme and trailing slash, for showing a URL the way people type it.
func bare(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	return strings.TrimSuffix(u, "/")
}
