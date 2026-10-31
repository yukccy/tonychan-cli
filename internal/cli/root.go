// Package cli is the tonychan command: yukccy.com in a terminal, one command per page.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/yukccy/tonychan-cli/internal/site"
)

// app is what every command needs. Tests build one with a fixed clock and plain style.
type app struct {
	client *site.Client
	out    io.Writer
	errOut io.Writer
	style  Style
	now    func() time.Time
	open   func(url string) error
}

// Execute runs the CLI and returns the process exit code.
func Execute(version string) int {
	a := &app{
		client: site.New(version),
		out:    os.Stdout,
		errOut: os.Stderr,
		style:  detectStyle(os.Stdout),
		now:    time.Now,
		open:   openBrowser,
	}
	cmd := a.rootCmd(version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "tonychan:", err)
		return 1
	}
	return 0
}

func (a *app) rootCmd(version string) *cobra.Command {
	var work, writing bool
	root := &cobra.Command{
		Use:   "tonychan",
		Short: "Tony Chan's site, yukccy.com, in a terminal",
		Long: "Tony Chan's site, yukccy.com, in a terminal.\n\n" +
			"With no command it shows the Home page: who I am, what I'm doing now, and the log of\n" +
			"new posts and changes in my work.",
		Example: "  tonychan\n  tonychan work\n  tonychan writing --all\n  tonychan writing private-kubernetes-access-with-tailscale",
		Version: version,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			idx, err := a.index(cmd.Context())
			if err != nil {
				return err
			}
			a.home(idx, logFilter{work: work, writing: writing})
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.Flags().BoolVar(&work, "work", false, "only work entries in the log")
	root.Flags().BoolVar(&writing, "writing", false, "only posts in the log")
	root.SetOut(a.out)
	root.SetErr(a.errOut)
	root.AddCommand(a.workCmd(), a.writingCmd(), a.openCmd(), a.contactCmd())
	return root
}

// index loads the site index. A stale cached copy is used with a warning, not an error.
func (a *app) index(ctx context.Context) (*site.Index, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	idx, err := a.client.Index(ctx)
	var stale *site.Stale
	if errors.As(err, &stale) {
		fmt.Fprintln(a.errOut, a.style.Dim("tonychan: "+stale.Error()))
		return idx, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not reach %s: %w", a.client.Base, err)
	}
	return idx, nil
}
