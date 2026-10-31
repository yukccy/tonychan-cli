package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yukccy/tonychan-cli/internal/site"
)

func (a *app) workCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "work",
		Short: "Roles, certifications and education (yukccy.com/work)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			idx, err := a.index(cmd.Context())
			if err != nil {
				return err
			}
			a.work(idx, all)
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "include expired certifications")
	return cmd
}

func (a *app) work(idx *site.Index, all bool) {
	s := a.style
	w := a.out
	wk := idx.Work

	head := []string{idx.Headline}
	if y := yearsSince(wk.Jobs, a.now()); y > 0 {
		head = append(head, fmt.Sprintf("%d years", y))
	}
	head = append(head, idx.Now.Place)
	fmt.Fprintln(w, s.Bold(joinNonEmpty(" · ", head...)))
	fmt.Fprintln(w)

	fmt.Fprintln(w, s.Label("Experience"))
	for _, j := range wk.Jobs {
		if j.Break {
			fmt.Fprintln(w, s.Dim("│ "+pad(j.Dates, 21)+" "+joinNonEmpty(", ", j.Title, j.Org)))
			continue
		}
		fmt.Fprintln(w, s.Red("*")+" "+pad(j.Dates, 21)+" "+joinNonEmpty(" · ", s.Bold(j.Title), j.Org, j.Place))
	}
	fmt.Fprintln(w)

	thisMonth := a.now().Format("2006-01")
	var valid, expired []site.Cert
	for _, c := range wk.Certifications {
		if c.Expires != "" && c.Expires < thisMonth {
			expired = append(expired, c)
		} else {
			valid = append(valid, c)
		}
	}
	nameWidth := 0
	for _, c := range wk.Certifications {
		nameWidth = max(nameWidth, len([]rune(c.Name)))
	}
	nameWidth = min(nameWidth, max(s.Width-24, 20))
	fmt.Fprintln(w, s.Label("Certifications"))
	for _, c := range valid {
		line := "  " + pad(truncate(c.Name, nameWidth), nameWidth) + "  " + c.Issued
		if c.Expires != "" {
			line += "  valid to " + c.Expires
		}
		fmt.Fprintln(w, line)
	}
	if all {
		for _, c := range expired {
			fmt.Fprintln(w, s.Dim("  "+pad(truncate(c.Name, nameWidth), nameWidth)+"  "+c.Issued+"  expired "+c.Expires))
		}
	} else if len(expired) > 0 {
		fmt.Fprintln(w, s.Dim(fmt.Sprintf("  + %d expired · tonychan work --all", len(expired))))
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, s.Label("Education"))
	for _, e := range wk.Education {
		fmt.Fprintln(w, "  "+joinNonEmpty(" · ", e.Degree, e.School)+"  "+e.Years)
	}
	fmt.Fprintln(w)
	url := strings.TrimRight(idx.Site, "/") + "/work/"
	fmt.Fprintln(w, field(s.Label("On the web"), "ON THE WEB", 12)+s.Link(url, bare(url)))
}

// yearsSince counts whole years from the start of the earliest role ("Jul 2019 – Jun 2020").
func yearsSince(jobs []site.Job, now time.Time) int {
	var first time.Time
	for _, j := range jobs {
		if j.Break {
			continue
		}
		start := strings.TrimSpace(strings.SplitN(j.Dates, "–", 2)[0])
		t, err := time.Parse("Jan 2006", start)
		if err != nil {
			continue
		}
		if first.IsZero() || t.Before(first) {
			first = t
		}
	}
	if first.IsZero() {
		return 0
	}
	years := now.Year() - first.Year()
	if now.Month() < first.Month() {
		years--
	}
	return years
}
