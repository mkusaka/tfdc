package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreHitMissTTLAndNoCache(t *testing.T) {
	t.Run("creates cache directory structure", func(t *testing.T) {
		dir := t.TempDir()
		_, err := NewStore(dir, time.Hour, true)
		if err != nil {
			t.Fatal(err)
		}

		for _, p := range []string{
			filepath.Join(dir, "v1", "meta.json"),
			filepath.Join(dir, "v1", "entries"),
			filepath.Join(dir, "v1", "tmp"),
		} {
			if _, err := os.Stat(p); err != nil {
				t.Fatalf("expected path to exist: %s (%v)", p, err)
			}
		}
	})

	t.Run("hit and ttl expire", func(t *testing.T) {
		dir := t.TempDir()
		store, err := NewStore(dir, time.Hour, true)
		if err != nil {
			t.Fatal(err)
		}

		now := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
		store.now = func() time.Time { return now }

		if err := store.Set("GET", "https://example.com/v2/provider-docs/1", 200, "application/json", []byte(`{"ok":true}`)); err != nil {
			t.Fatal(err)
		}

		b, ok, err := store.Get("GET", "https://example.com/v2/provider-docs/1")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("expected cache hit")
		}
		if string(b) != `{"ok":true}` {
			t.Fatalf("unexpected body: %s", string(b))
		}

		store.now = func() time.Time { return now.Add(2 * time.Hour) }
		_, ok, err = store.Get("GET", "https://example.com/v2/provider-docs/1")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatalf("expected cache miss after ttl expiration")
		}
	})

	t.Run("no-cache mode", func(t *testing.T) {
		dir := t.TempDir()
		store, err := NewStore(dir, time.Hour, false)
		if err != nil {
			t.Fatal(err)
		}

		if err := store.Set("GET", "https://example.com/a", 200, "text/plain", []byte("x")); err != nil {
			t.Fatal(err)
		}
		_, ok, err := store.Get("GET", "https://example.com/a")
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatalf("expected miss in no-cache mode")
		}

		v1Dir := filepath.Join(dir, "v1")
		if _, err := os.Stat(v1Dir); !os.IsNotExist(err) {
			t.Fatalf("expected no cache directory in no-cache mode")
		}
	})
}
