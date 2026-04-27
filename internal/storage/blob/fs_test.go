package blob

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestFSStore_PutOpenDelete(t *testing.T) {
	s, err := NewFSStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFSStore: %v", err)
	}
	ctx := context.Background()

	want := []byte("hello attachment")
	if err := s.Put(ctx, "abc123", "text/plain", int64(len(want)), bytes.NewReader(want)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, err := s.Open(ctx, "abc123")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Open returned %q, want %q", got, want)
	}

	if err := s.Delete(ctx, "abc123"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Open(ctx, "abc123"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Open after delete: err = %v, want ErrNotFound", err)
	}
}

func TestFSStore_Open_MissingReturnsErrNotFound(t *testing.T) {
	s, _ := NewFSStore(t.TempDir())
	_, err := s.Open(context.Background(), "never-existed")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestFSStore_Delete_MissingIsNotError(t *testing.T) {
	s, _ := NewFSStore(t.TempDir())
	if err := s.Delete(context.Background(), "never-existed"); err != nil {
		t.Fatalf("Delete missing returned %v, want nil", err)
	}
}

func TestFSStore_RejectsBadKeys(t *testing.T) {
	s, _ := NewFSStore(t.TempDir())
	ctx := context.Background()

	for _, k := range []string{"", "../escape", "..", "a/b", `a\b`} {
		if err := s.Put(ctx, k, "", 0, strings.NewReader("x")); err == nil {
			t.Errorf("Put(%q) accepted an unsafe key", k)
		}
		if _, err := s.Open(ctx, k); err == nil {
			t.Errorf("Open(%q) accepted an unsafe key", k)
		}
	}
}

func TestNewFSStore_RequiresBaseDir(t *testing.T) {
	if _, err := NewFSStore(""); err == nil {
		t.Fatal("expected error for empty base dir")
	}
}
