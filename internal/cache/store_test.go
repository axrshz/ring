package cache

import "testing"

func TestStoreGetSetDelete(t *testing.T) {
	store := NewStore()

	if _, err := store.Get("missing"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for missing key, got %v", err)
	}

	store.Set("language", "go")

	value, err := store.Get("language")
	if err != nil {
		t.Fatalf("expected stored value, got error %v", err)
	}
	if value != "go" {
		t.Fatalf("expected go, got %q", value)
	}

	store.Delete("language")

	if _, err := store.Get("language"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
