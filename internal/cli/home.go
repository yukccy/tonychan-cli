package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/yukccy/tonychan-cli/internal/site"
)

// banner is "TONY CHAN." in block letters. The last three cells of the last two lines are the
// full stop, drawn in red like the site's.
var banner = []string{
	"████████╗ ██████╗ ███╗   ██╗██╗   ██╗   ██████╗██╗  ██╗ █████╗ ███╗   ██╗",
	"╚══██╔══╝██╔═══██╗████╗  ██║╚██╗ ██╔╝  ██╔════╝██║  ██║██╔══██╗████╗  ██║",
	"   ██║   ██║   ██║██╔██╗ ██║ ╚████╔╝   ██║     ███████║███████║██╔██╗ ██║",
	"   ██║   ██║   ██║██║╚██╗██║  ╚██╔╝    ██║     ██╔══██║██╔══██║██║╚██╗██║",
	"   ██║   ╚██████╔╝██║ ╚████║   ██║     ╚██████╗██║  ██║██║  ██║██║ ╚████║██╗",
	"   ╚═╝    ╚═════╝ ╚═╝  ╚═══╝   ╚═╝      ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝",
}

// homeLogLength matches the site's homeLogLength.
const homeLogLength = 10

type logFilter struct{ work, writing bool }

type logRow struct {
	date  string // sorts the log; "YYYY-MM-DD"
	label string // shown in the WHEN column
	title string
	url   string
	post  bool
}

func (a *app) home(idx *site.Index, f logFilter) {
	s := a.style
	w := a.out

	a.printBanner(idx.Name)
	fmt.Fprintln(w)
	fmt.Fprintln(w, wrap(intro(idx), min(s.Width, 100)))
	fmt.Fprintln(w)

	const key = 13
	now := idx.Now
	fmt.Fprintf(w, "%s%s\n", field(s.Label("Now"), "NOW", key), joinNonEmpty(" · ",
		now.Role+", "+now.Org, "since "+now.Since, now.Place))
	if len(idx.Posts) > 0 {
		p := idx.Posts[0]
		fmt.Fprintf(w, "%s%s · %s\n", field(s.Label("Latest post"), "LATEST POST", key), s.Link(p.URL, p.Title), p.Date)
	}
	fmt.Fprintln(w)

	rows := homeLog(idx, f)
	heading := "Log"
	switch {
	case f.work && !f.writing:
		heading = "Log · work"
	case f.writing && !f.work:
		heading = "Log · writing"
	}
	fmt.Fprintln(w, s.Label(heading))
	for _, r := range rows {
		fmt.Fprintln(w, a.logLine(r))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, s.Dim("work · writing --all · writing <slug> · open · contact · --help"))
	c := idx.Contact
	fmt.Fprintln(w, s.Dim(joinNonEmpty(" · ",
		s.Link("mailto:"+c.Email, c.Email), s.Link(c.GitHub, bare(c.GitHub)), s.Link(c.LinkedIn, bare(c.LinkedIn)))))
}

func (a *app) printBanner(name string) {
	s := a.style
	if s.Width < runewidth.StringWidth(banner[4]) {
		// Too narrow for the block letters: the name, with the red full stop.
		fmt.Fprintln(a.out, s.Bold(name)+s.Red("."))
		return
	}
	for i, line := range banner {
		if i >= 4 {
			r := []rune(line)
			line = string(r[:len(r)-3]) + s.Red(string(r[len(r)-3:]))
		}
		fmt.Fprintln(a.out, line)
	}
}

// intro is the one-line introduction from the site, or one built from the index if the site
// does not publish it.
func intro(idx *site.Index) string {
	if idx.Intro != "" {
		return idx.Intro
	}
	return fmt.Sprintf("%s at %s, in %s.", idx.Headline, idx.Now.Org, firstPart(idx.Now.Place))
}

// homeLog merges posts and work events, newest first, like the site's `git log --all`.
func homeLog(idx *site.Index, f logFilter) []logRow {
	both := f.work == f.writing
	var rows []logRow
	if both || f.writing {
		for _, p := range idx.Posts {
			rows = append(rows, logRow{date: p.Date, label: p.Date, title: p.Title, url: p.URL, post: true})
		}
	}
	if both || f.work {
		for _, e := range idx.Log {
			rows = append(rows, logRow{date: e.Date, label: e.Label, title: e.Title})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].date > rows[j].date })
	if len(rows) > homeLogLength {
		rows = rows[:homeLogLength]
	}
	return rows
}

// logLine draws one log row on two lanes: work on the left (*), writing on the right (●).
func (a *app) logLine(r logRow) string {
	s := a.style
	lanes, kind := "* │", "WORK   "
	if r.post {
		lanes, kind = "│ "+s.Red("●"), s.Red("WRITING")
	}
	prefix := lanes + "  " + pad(r.label, 10) + "  " + kind + "  "
	room := s.Width - 3 - 2 - 10 - 2 - 7 - 2
	title := truncate(r.title, max(room, 10))
	if r.post {
		title = s.Link(r.url, title)
	}
	return prefix + title
}

func joinNonEmpty(sep string, parts ...string) string {
	var keep []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			keep = append(keep, p)
		}
	}
	return strings.Join(keep, sep)
}

func firstPart(place string) string {
	if i := strings.Index(place, ","); i >= 0 {
		return place[:i]
	}
	return place
}

// wrap breaks text into lines of at most width cells, on spaces.
func wrap(text string, width int) string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case runewidth.StringWidth(line)+1+runewidth.StringWidth(word) > width:
			lines = append(lines, line)
			line = word
		default:
			line += " " + word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
