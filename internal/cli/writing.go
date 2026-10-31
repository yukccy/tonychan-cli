package cli

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/yukccy/tonychan-cli/internal/site"
)

// writingDefault is how many posts `tonychan writing` lists without --all.
const writingDefault = 5

func (a *app) writingCmd() *cobra.Command {
	var all, noPager bool
	var tag string
	cmd := &cobra.Command{
		Use:   "writing [slug or URL]",
		Short: "Posts, newest first, or one post in full (yukccy.com/writing)",
		Long: "With no argument, lists posts newest first. With a slug or a yukccy.com/writing URL,\n" +
			"shows that post in the terminal, paged.",
		Example: "  tonychan writing --all\n  tonychan writing --tag kubernetes\n" +
			"  tonychan writing private-kubernetes-access-with-tailscale\n" +
			"  tonychan writing https://yukccy.com/writing/why-do-we-need-helm/",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := a.index(cmd.Context())
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return a.listPosts(idx, all, tag)
			}
			p, err := findPost(idx, args[0])
			if err != nil {
				return err
			}
			md, err := a.client.Markdown(cmd.Context(), idx, p)
			var stale *site.Stale
			if errors.As(err, &stale) {
				fmt.Fprintln(a.errOut, a.style.Dim("tonychan: "+stale.Error()))
			} else if err != nil {
				return fmt.Errorf("could not fetch %q: %w", p.Slug, err)
			}
			return a.readPost(md, noPager)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "list every post")
	cmd.Flags().StringVar(&tag, "tag", "", "only posts with this topic, e.g. kubernetes")
	cmd.Flags().BoolVar(&noPager, "no-pager", false, "print the post instead of paging it")
	return cmd
}

func (a *app) listPosts(idx *site.Index, all bool, tag string) error {
	s := a.style
	w := a.out
	posts := idx.Posts
	if tag != "" {
		posts = withTag(posts, tag)
		if len(posts) == 0 {
			return fmt.Errorf("no posts tagged %q", tag)
		}
	}
	shown := posts
	if !all && tag == "" && len(posts) > writingDefault {
		shown = posts[:writingDefault]
	}

	fmt.Fprintln(w, s.Label("Date")+"        "+s.Label("Title"))
	for _, p := range shown {
		title := truncate(p.Title, max(s.Width-12, 20))
		fmt.Fprintln(w, p.Date+"  "+s.Bold(title))
		fmt.Fprintln(w, "            "+s.Link(p.URL, s.Dim(truncate(bare(p.URL), max(s.Width-12, 20)))))
	}
	fmt.Fprintln(w)
	var hint []string
	if more := len(posts) - len(shown); more > 0 {
		hint = append(hint, fmt.Sprintf("… %d more · tonychan writing --all", more))
	}
	if s.Links {
		hint = append(hint, "links open in the browser")
	}
	hint = append(hint, "tonychan writing <slug or URL> reads one here")
	fmt.Fprintln(w, s.Dim(strings.Join(hint, " · ")))
	return nil
}

func withTag(posts []site.Post, tag string) []site.Post {
	tag = strings.ToLower(strings.TrimPrefix(tag, "#"))
	var out []site.Post
	for _, p := range posts {
		for _, t := range p.Tags {
			if strings.ToLower(t) == tag {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// findPost accepts a slug, a yukccy.com/writing URL (any host, with or without scheme), or the
// post's Medium URL. A unique prefix of a slug also works.
func findPost(idx *site.Index, arg string) (site.Post, error) {
	key := strings.TrimSpace(arg)
	for _, p := range idx.Posts {
		if key == p.MediumURL && key != "" {
			return p, nil
		}
	}
	key = slugFromArg(key)
	var prefixed []site.Post
	for _, p := range idx.Posts {
		if p.Slug == key {
			return p, nil
		}
		if strings.HasPrefix(p.Slug, key) {
			prefixed = append(prefixed, p)
		}
	}
	if len(prefixed) == 1 {
		return prefixed[0], nil
	}
	if len(prefixed) > 1 {
		var names []string
		for _, p := range prefixed {
			names = append(names, "  "+p.Slug)
		}
		return site.Post{}, fmt.Errorf("%q matches more than one post:\n%s", arg, strings.Join(names, "\n"))
	}
	return site.Post{}, fmt.Errorf("no post %q; see tonychan writing --all", arg)
}

func slugFromArg(arg string) string {
	s := arg
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil {
			s = u.Path
		}
	}
	if i := strings.Index(s, "/writing/"); i >= 0 {
		s = s[i+len("/writing/"):]
	}
	s = strings.TrimSuffix(strings.Trim(s, "/"), "/index.md")
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	return strings.ToLower(s)
}

func (a *app) readPost(md string, noPager bool) error {
	s := a.style
	width := min(s.Width, 100)
	style := "notty"
	if s.Color {
		style = "dark"
		if !termIsDark() {
			style = "light"
		}
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(width-4))
	if err != nil {
		return err
	}
	out, err := r.Render(terminalImages(md))
	if err != nil {
		return err
	}
	if !s.Color {
		out = trimLines(out)
	}
	if noPager || !isTerminal(a.out) {
		_, err = io.WriteString(a.out, out)
		return err
	}
	return page(out, a.out)
}

var mdImage = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)

// terminalImages swaps images, which a terminal can't show, for their alt text.
func terminalImages(md string) string {
	return mdImage.ReplaceAllStringFunc(md, func(m string) string {
		alt := strings.TrimSpace(mdImage.FindStringSubmatch(m)[1])
		if alt == "" {
			return "(image)"
		}
		return "(image: " + alt + ")"
	})
}

// trimLines drops the trailing spaces glamour pads every line with.
func trimLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

// page sends text through $PAGER (default `less -RFX`: colours kept, quits if it fits).
func page(text string, out io.Writer) error {
	pager := os.Getenv("TONYCHAN_PAGER")
	if pager == "" {
		pager = os.Getenv("PAGER")
	}
	if pager == "" {
		if runtime.GOOS == "windows" {
			_, err := io.WriteString(out, text)
			return err
		}
		pager = "less -RFX"
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", pager)
	} else {
		cmd = exec.Command("sh", "-c", pager)
	}
	cmd.Stdin = strings.NewReader(text)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil // the pager ran and the user quit it
		}
		_, err = io.WriteString(out, text) // no usable pager
		return err
	}
	return nil
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// termIsDark guesses the background from COLORFGBG ("15;0" is light text on dark), defaulting
// to dark, which most terminals are.
func termIsDark() bool {
	v := os.Getenv("COLORFGBG")
	if i := strings.LastIndex(v, ";"); i >= 0 {
		switch v[i+1:] {
		case "7", "15":
			return false
		}
	}
	return true
}
