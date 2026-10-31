package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yukccy/tonychan-cli/internal/site"
)

const testMarkdown = "# Why do we need Helm?\n\n2024-07-07 · 3 min read\n\n![Banner](https://yukccy.com/writing/why-do-we-need-helm/a.png)\n\nHelm is a package manager.\n"

// run executes the CLI against a local copy of the site with a fixed clock and plain output.
func run(t *testing.T, width int, args ...string) (string, error) {
	t.Helper()
	index, err := os.ReadFile("testdata/index.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/index.json":
			w.Write(index)
		case strings.HasSuffix(r.URL.Path, "/index.md"):
			w.Write([]byte(testMarkdown))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	var out, errOut bytes.Buffer
	a := &app{
		client: &site.Client{Base: srv.URL, HTTP: srv.Client()},
		out:    &out,
		errOut: &errOut,
		style:  Style{Width: width},
		now:    func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) },
		open:   func(string) error { return nil },
	}
	cmd := a.rootCmd("test")
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), err
}

func mustRun(t *testing.T, width int, args ...string) string {
	t.Helper()
	out, err := run(t, width, args...)
	if err != nil {
		t.Fatalf("tonychan %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func lines(out, prefix string) []string {
	var keep []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, prefix) {
			keep = append(keep, l)
		}
	}
	return keep
}

func TestHome(t *testing.T) {
	out := mustRun(t, 100)
	for _, want := range []string{
		banner[0],
		"NOW          Member of Technical Staff, OpusClip · since Mar 2026",
		"LATEST POST  Private Kubernetes Access with Tailscale · 2025-10-15",
		"* │  2026-03     WORK     Joined OpusClip as Member of Technical Staff",
		"│ ●  2025-10-15  WRITING  Private Kubernetes Access with Tailscale",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("home is missing %q\n%s", want, out)
		}
	}
	if n := len(lines(out, "* │")) + len(lines(out, "│ ●")); n != homeLogLength {
		t.Errorf("log has %d rows, want %d", n, homeLogLength)
	}
}

func TestHomeFilters(t *testing.T) {
	work := mustRun(t, 100, "--work")
	if len(lines(work, "│ ●")) != 0 || len(lines(work, "* │")) == 0 {
		t.Errorf("--work should list only work entries:\n%s", work)
	}
	writing := mustRun(t, 100, "--writing")
	if len(lines(writing, "* │")) != 0 || len(lines(writing, "│ ●")) != homeLogLength {
		t.Errorf("--writing should list only posts:\n%s", writing)
	}
}

func TestHomeNarrowTerminal(t *testing.T) {
	out := mustRun(t, 40)
	if strings.Contains(out, banner[0]) || !strings.HasPrefix(out, "Tony Chan.\n") {
		t.Errorf("a narrow terminal should get the plain name, got:\n%s", out)
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "│ ●") && len([]rune(l)) > 40 {
			t.Errorf("log row overflows 40 columns: %q", l)
		}
	}
}

func TestWork(t *testing.T) {
	out := mustRun(t, 100, "work")
	for _, want := range []string{"DevOps and Platform Engineer · 7 years", "│ Apr 2022 – Jun 2022", "+ 3 expired", "valid to 2027-06"} {
		if !strings.Contains(out, want) {
			t.Errorf("work is missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "expired 2026-05") {
		t.Error("expired certifications shown without --all")
	}
	all := mustRun(t, 100, "work", "--all")
	if !strings.Contains(all, "AWS Certified Security – Specialty") || !strings.Contains(all, "expired 2026-05") {
		t.Errorf("--all should list expired certifications:\n%s", all)
	}
}

func TestYearsSince(t *testing.T) {
	jobs := []site.Job{{Dates: "Jul 2019 – Jun 2020"}, {Dates: "Apr 2022 – Jun 2022", Break: true}, {Dates: "Mar 2026 – now"}}
	for _, tc := range []struct {
		now  time.Time
		want int
	}{
		{time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), 6},
		{time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), 7},
		{time.Date(2019, 7, 1, 0, 0, 0, 0, time.UTC), 0},
	} {
		if got := yearsSince(jobs, tc.now); got != tc.want {
			t.Errorf("yearsSince at %s = %d, want %d", tc.now.Format("2006-01"), got, tc.want)
		}
	}
}

func TestWritingList(t *testing.T) {
	out := mustRun(t, 100, "writing")
	if n := len(lines(out, "20")); n != writingDefault {
		t.Errorf("listed %d posts, want %d:\n%s", n, writingDefault, out)
	}
	if !strings.Contains(out, "… 9 more · tonychan writing --all") {
		t.Errorf("missing the more hint:\n%s", out)
	}
	if !strings.Contains(out, "            yukccy.com/writing/private-kubernetes-access-with-tailscale\n") {
		t.Errorf("posts should show their URL:\n%s", out)
	}
	if n := len(lines(mustRun(t, 100, "writing", "--all"), "20")); n != 14 {
		t.Errorf("--all listed %d posts, want 14", n)
	}
	helm := mustRun(t, 100, "writing", "--tag", "#Helm")
	if n := len(lines(helm, "20")); n != 3 {
		t.Errorf("--tag helm listed %d posts, want 3:\n%s", n, helm)
	}
	if _, err := run(t, 100, "writing", "--tag", "cobol"); err == nil {
		t.Error("an unknown tag should be an error")
	}
}

func TestWritingRead(t *testing.T) {
	out := mustRun(t, 100, "writing", "why-do-we-need-helm")
	if !strings.Contains(out, "Helm is a package manager.") || !strings.Contains(out, "(image: Banner)") {
		t.Errorf("post not rendered:\n%s", out)
	}
	if strings.Contains(out, "a.png") {
		t.Errorf("image URLs should be replaced by their alt text:\n%s", out)
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.HasSuffix(l, " ") {
			t.Errorf("trailing spaces in plain output: %q", l)
		}
	}
}

func TestFindPost(t *testing.T) {
	idx := &site.Index{Posts: []site.Post{
		{Slug: "why-do-we-need-helm", MediumURL: "https://medium.com/@yukccy/why-do-we-need-helm-abc123"},
		{Slug: "my-top-3-cncf-projects-in-devops"},
		{Slug: "my-homelab"},
	}}
	for arg, want := range map[string]string{
		"why-do-we-need-helm":                                     "why-do-we-need-helm",
		"https://yukccy.com/writing/why-do-we-need-helm/":         "why-do-we-need-helm",
		"yukccy.com/writing/why-do-we-need-helm":                  "why-do-we-need-helm",
		"https://yukccy.com/writing/why-do-we-need-helm/index.md": "why-do-we-need-helm",
		"https://medium.com/@yukccy/why-do-we-need-helm-abc123":   "why-do-we-need-helm",
		"why":        "why-do-we-need-helm",
		"MY-TOP":     "my-top-3-cncf-projects-in-devops",
		"my-homelab": "my-homelab",
	} {
		p, err := findPost(idx, arg)
		if err != nil || p.Slug != want {
			t.Errorf("findPost(%q) = %q, %v; want %q", arg, p.Slug, err, want)
		}
	}
	for _, arg := range []string{"my", "nope", ""} {
		if p, err := findPost(idx, arg); err == nil {
			t.Errorf("findPost(%q) = %q, want an error", arg, p.Slug)
		}
	}
}

func TestStyleOff(t *testing.T) {
	s := Style{Width: 80}
	if s.Red("x") != "x" || s.Link("https://a", "x") != "x" {
		t.Error("plain style must not emit escape codes")
	}
	on := Style{Color: true, Links: true, Width: 80}
	if !strings.Contains(on.Link("https://a", "x"), "\x1b]8;;https://a") {
		t.Error("links should be OSC 8 hyperlinks")
	}
	if got := field(on.Bold("NOW"), "NOW", 6); !strings.HasSuffix(got, "\x1b[0m   ") {
		t.Errorf("field should pad by visible width, got %q", got)
	}
}
