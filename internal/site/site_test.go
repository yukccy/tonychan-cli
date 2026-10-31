package site

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIndexCachesAndFallsBack(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "tonychan/test" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		w.Write([]byte(`{"name":"Tony Chan","site":"https://yukccy.com/","posts":[{"slug":"a","markdown":"https://yukccy.com/writing/a/index.md"}]}`))
	}))
	c := &Client{Base: srv.URL, CacheDir: t.TempDir(), UserAgent: "tonychan/test", HTTP: srv.Client()}

	idx, err := c.Index(context.Background())
	if err != nil || idx.Name != "Tony Chan" {
		t.Fatalf("online: %+v, %v", idx, err)
	}

	srv.Close()
	idx, err = c.Index(context.Background())
	var stale *Stale
	if !errors.As(err, &stale) {
		t.Fatalf("offline with a cache should return *Stale, got %v", err)
	}
	if idx == nil || idx.Name != "Tony Chan" {
		t.Fatalf("offline should return the cached index, got %+v", idx)
	}

	c.CacheDir = t.TempDir()
	if idx, err = c.Index(context.Background()); err == nil || errors.As(err, &stale) || idx != nil {
		t.Fatalf("offline without a cache should fail, got %+v, %v", idx, err)
	}
}

func TestRebase(t *testing.T) {
	idx := &Index{Site: "https://yukccy.com/"}
	c := &Client{Base: "http://localhost:1313"}
	if got := c.Rebase(idx, "https://yukccy.com/writing/a/index.md"); got != "http://localhost:1313/writing/a/index.md" {
		t.Errorf("Rebase = %q", got)
	}
	c.Base = "https://yukccy.com"
	if got := c.Rebase(idx, "https://yukccy.com/writing/a/index.md"); got != "https://yukccy.com/writing/a/index.md" {
		t.Errorf("Rebase onto the same site changed the URL: %q", got)
	}
}
