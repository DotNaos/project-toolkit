package syncer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DotNaos/repo-kit/cli/internal/checker"
	"github.com/DotNaos/repo-kit/cli/internal/config"
	"github.com/DotNaos/repo-kit/cli/internal/lock"
)

func TestSyncAndCheckDrift(t *testing.T) {
	root := t.TempDir()
	kitRoot := filepath.Join(root, "kit")
	target := filepath.Join(root, "target")

	mustMkdir(t, filepath.Join(kitRoot, "assets", "templates"))
	mustMkdir(t, filepath.Join(kitRoot, "manifests"))
	mustMkdir(t, target)

	mustWrite(t, filepath.Join(kitRoot, "assets", "templates", "hello.txt"), []byte("hello\n"))
	mustWrite(t, filepath.Join(kitRoot, "manifests", "default.yaml"), []byte("name: default\nfiles:\n  - source: templates/hello.txt\n    target: hello.txt\n"))

	if err := config.Write(target, config.RepositoryConfig{Version: 1, Manifest: "default", SourceURL: "https://example.test/repo-kit"}); err != nil {
		t.Fatal(err)
	}
	if err := Sync(target, kitRoot); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(target, "hello.txt")); err != nil {
		t.Fatal(err)
	}
	l, err := lock.Read(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Files) != 1 || l.Files[0].Target != "hello.txt" {
		t.Fatalf("unexpected lock: %#v", l)
	}

	drift, err := checker.Drift(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 0 {
		t.Fatalf("expected no drift, got %v", drift)
	}

	mustWrite(t, filepath.Join(target, "hello.txt"), []byte("changed\n"))
	drift, err = checker.Drift(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(drift) != 1 {
		t.Fatalf("expected drift, got %v", drift)
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
