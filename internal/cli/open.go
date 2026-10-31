package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

func (a *app) openCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open [slug or URL]",
		Short: "Open the site, or a post, in the browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := a.index(cmd.Context())
			if err != nil {
				return err
			}
			target := idx.Site
			if len(args) == 1 {
				p, err := findPost(idx, args[0])
				if err != nil {
					return err
				}
				target = p.URL
			}
			fmt.Fprintln(a.out, target)
			return a.open(target)
		},
	}
}

func (a *app) contactCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "contact",
		Short: "Email and profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			idx, err := a.index(cmd.Context())
			if err != nil {
				return err
			}
			s := a.style
			c := idx.Contact
			rows := []struct{ label, url, text string }{
				{"Email", "mailto:" + c.Email, c.Email},
				{"GitHub", c.GitHub, bare(c.GitHub)},
				{"LinkedIn", c.LinkedIn, bare(c.LinkedIn)},
				{"Medium", c.Medium, bare(c.Medium)},
				{"Site", idx.Site, bare(idx.Site)},
			}
			for _, r := range rows {
				if r.text == "" {
					continue
				}
				fmt.Fprintln(a.out, field(s.Label(r.label), r.label, 10)+s.Link(r.url, r.text))
			}
			return nil
		},
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
