package completioncache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreFreshStaleAndExpired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "completion-cache.json")
	store := NewStore(path)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	key := Key("profile", "https://example.test/api/v1/", "demo", "issues")
	if err := store.Put(key, []string{"2\tSecond", "1\tFirst"}); err != nil {
		t.Fatal(err)
	}
	values, fresh, ok := store.Get(key)
	if !ok || !fresh || len(values) != 2 || values[0] != "1\tFirst" {
		t.Fatalf("fresh values=%#v fresh=%t ok=%t", values, fresh, ok)
	}
	store.now = func() time.Time { return now.Add(FreshTTL + time.Second) }
	_, fresh, ok = store.Get(key)
	if !ok || fresh {
		t.Fatalf("stale fresh=%t ok=%t", fresh, ok)
	}
	store.now = func() time.Time { return now.Add(StaleTTL + time.Second) }
	_, _, ok = store.Get(key)
	if ok {
		t.Fatal("expired cache entry remained available")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
}

func TestStoreIgnoresCorruptAndWrongVersionFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "completion-cache.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(path)
	if _, _, ok := store.Get("missing"); ok {
		t.Fatal("corrupt cache returned an entry")
	}
	if err := store.Put("key", []string{"value"}); err != nil {
		t.Fatal(err)
	}
	if values, fresh, ok := store.Get("key"); !ok || !fresh || len(values) != 1 {
		t.Fatalf("values=%#v fresh=%t ok=%t", values, fresh, ok)
	}
}

func TestKeyScopesProfileAPIProjectAndKind(t *testing.T) {
	base := Key("p", "https://example.test/api/v1/", "demo", "issues")
	for _, key := range []string{
		Key("other", "https://example.test/api/v1/", "demo", "issues"),
		Key("p", "https://other.test/api/v1/", "demo", "issues"),
		Key("p", "https://example.test/api/v1/", "other", "issues"),
		Key("p", "https://example.test/api/v1/", "demo", "stories"),
	} {
		if key == base {
			t.Fatal("cache key scope collision")
		}
	}
}
