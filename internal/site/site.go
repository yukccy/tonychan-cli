// Package site reads what yukccy.com publishes for this CLI: /index.json (work, log and the
// post list) and each post's Markdown. Both are cached on disk, so the CLI still works offline
// with the last copy it saw.
package site

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultBase is the site the CLI reads. TONYCHAN_SITE overrides it, e.g. for a local hugo server.
const DefaultBase = "https://yukccy.com"

type Index struct {
	Name     string  `json:"name"`
	Site     string  `json:"site"`
	Headline string  `json:"headline"`
	Intro    string  `json:"intro"`
	Now      Now     `json:"now"`
	Work     Work    `json:"work"`
	Log      []Event `json:"log"`
	Posts    []Post  `json:"posts"`
	Contact  Contact `json:"contact"`
}

type Now struct {
	Org   string `json:"org"`
	Place string `json:"place"`
	Role  string `json:"role"`
	Since string `json:"since"`
}

type Work struct {
	Jobs           []Job       `json:"jobs"`
	Certifications []Cert      `json:"certifications"`
	Education      []Education `json:"education"`
}

// Job is one role. Break marks a gap between roles, such as a relocation.
type Job struct {
	Dates string `json:"dates"`
	Title string `json:"title"`
	Org   string `json:"org"`
	Place string `json:"place"`
	Break bool   `json:"break"`
}

// Cert dates are "YYYY-MM". Expires is empty for certifications that do not expire.
type Cert struct {
	Name    string `json:"name"`
	Issuer  string `json:"issuer"`
	Issued  string `json:"issued"`
	Expires string `json:"expires"`
}

type Education struct {
	Degree string `json:"degree"`
	School string `json:"school"`
	Years  string `json:"years"`
}

// Event is a work entry in the Home log. Date is "YYYY-MM-DD"; Label is how it is shown.
type Event struct {
	Date  string `json:"date"`
	Label string `json:"label"`
	Title string `json:"title"`
	Note  string `json:"note"`
}

type Post struct {
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	Date        string   `json:"date"`
	URL         string   `json:"url"`
	Markdown    string   `json:"markdown"`
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
	ReadingTime int      `json:"reading_time"`
	MediumURL   string   `json:"medium_url"`
	SummaryOnly bool     `json:"summary_only"`
}

type Contact struct {
	Email    string `json:"email"`
	GitHub   string `json:"github"`
	LinkedIn string `json:"linkedin"`
	Medium   string `json:"medium"`
}

// Stale is returned alongside cached data when the site could not be reached.
type Stale struct {
	Saved time.Time
	Err   error
}

func (s *Stale) Error() string {
	return fmt.Sprintf("offline, showing the copy from %s (%v)", s.Saved.Format("2006-01-02"), s.Err)
}

func (s *Stale) Unwrap() error { return s.Err }

type Client struct {
	Base      string
	CacheDir  string // empty disables the cache
	UserAgent string
	HTTP      *http.Client
}

// New returns a client for TONYCHAN_SITE, or yukccy.com, caching under the user cache directory.
func New(version string) *Client {
	base := os.Getenv("TONYCHAN_SITE")
	if base == "" {
		base = DefaultBase
	}
	c := &Client{
		Base:      strings.TrimRight(base, "/"),
		UserAgent: "tonychan/" + version + " (+https://github.com/yukccy/tonychan-cli)",
		HTTP:      &http.Client{Timeout: 5 * time.Second},
	}
	if dir, err := os.UserCacheDir(); err == nil {
		c.CacheDir = filepath.Join(dir, "tonychan")
	}
	return c
}

// Index fetches /index.json. If the site is unreachable but a cached copy exists, it returns
// that copy and a *Stale error, which callers should show as a warning rather than fail on.
func (c *Client) Index(ctx context.Context) (*Index, error) {
	body, err := c.fetch(ctx, c.Base+"/index.json", "index.json")
	if body == nil {
		return nil, err
	}
	var idx Index
	if jerr := json.Unmarshal(body, &idx); jerr != nil {
		return nil, fmt.Errorf("reading index.json: %w", jerr)
	}
	return &idx, err
}

// Markdown fetches a post's Markdown, with the same offline fallback as Index.
func (c *Client) Markdown(ctx context.Context, idx *Index, p Post) (string, error) {
	body, err := c.fetch(ctx, c.Rebase(idx, p.Markdown), filepath.Join("posts", p.Slug+".md"))
	if body == nil {
		return "", err
	}
	return string(body), err
}

// Rebase points a URL from the index at the configured base, so TONYCHAN_SITE also covers the
// post links the index carries (they are always absolute yukccy.com URLs).
func (c *Client) Rebase(idx *Index, u string) string {
	site := strings.TrimRight(idx.Site, "/")
	if site != "" && site != c.Base && strings.HasPrefix(u, site) {
		return c.Base + strings.TrimPrefix(u, site)
	}
	return u
}

func (c *Client) fetch(ctx context.Context, url, cacheName string) ([]byte, error) {
	body, err := c.get(ctx, url)
	if err == nil {
		c.save(cacheName, body)
		return body, nil
	}
	if cached, saved, ok := c.load(cacheName); ok {
		return cached, &Stale{Saved: saved, Err: err}
	}
	return nil, err
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

// ErrNotFound means the site answered 404.
var ErrNotFound = errors.New("not found on the site")

func (c *Client) save(name string, body []byte) {
	if c.CacheDir == "" {
		return
	}
	path := filepath.Join(c.CacheDir, name)
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	// Write then rename, so a crash never leaves half a file behind.
	tmp := path + ".tmp"
	if os.WriteFile(tmp, body, 0o644) == nil {
		_ = os.Rename(tmp, path)
	}
}

func (c *Client) load(name string) ([]byte, time.Time, bool) {
	if c.CacheDir == "" {
		return nil, time.Time{}, false
	}
	path := filepath.Join(c.CacheDir, name)
	info, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}, false
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, false
	}
	return body, info.ModTime(), true
}
