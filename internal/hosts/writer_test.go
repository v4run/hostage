package hosts_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/v4run/hostage/internal/hosts"
)

func TestWriteAndRead(t *testing.T) {
	f, err := os.CreateTemp("", "hosts-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("127.0.0.1 localhost\n")
	f.Close()

	mtime, err := hosts.ReadMtime(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	lines := []hosts.Line{
		{Type: hosts.LineEntry, IP: "127.0.0.1", Hostnames: []string{"localhost"}},
	}
	err = hosts.Write(f.Name(), lines, mtime)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, _ := os.ReadFile(f.Name())
	if string(content) != "127.0.0.1 localhost\n" {
		t.Errorf("unexpected content: %q", string(content))
	}
}

func TestWriteConflict(t *testing.T) {
	f, err := os.CreateTemp("", "hosts-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("127.0.0.1 localhost\n")
	f.Close()

	stale := time.Now().Add(-10 * time.Second)

	lines := []hosts.Line{}
	err = hosts.Write(f.Name(), lines, stale)
	if err != hosts.ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestWriteAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts")
	os.WriteFile(path, []byte("127.0.0.1 localhost\n"), 0644)

	mtime, _ := hosts.ReadMtime(path)
	lines := []hosts.Line{
		{Type: hosts.LineEntry, IP: "10.0.0.1", Hostnames: []string{"newhost"}},
	}
	if err := hosts.Write(path, lines, mtime); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(path)
	if string(content) != "10.0.0.1 newhost\n" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestWritePreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts")
	original := "# header comment\n127.0.0.1 localhost\n# another comment\n"
	os.WriteFile(path, []byte(original), 0644)

	mtime, _ := hosts.ReadMtime(path)
	parsed := hosts.Parse(original)
	if err := hosts.Write(path, parsed, mtime); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(path)
	if string(content) != original {
		t.Errorf("comments not preserved:\nwant: %q\ngot:  %q", original, string(content))
	}
}
