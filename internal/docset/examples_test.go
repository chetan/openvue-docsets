package docset

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExampleAssetsDoNotReplaceReference(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	for _, dir := range []string{source, target} {
		if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("existing"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyExamples(context.Background(), source, target); err == nil {
		t.Fatal("replaced reference index")
	}
}

func TestExampleAssetsRejectSymlinks(t *testing.T) {
	source := t.TempDir()
	if err := os.Symlink("/etc/hosts", filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}
	if err := copyExamples(context.Background(), source, t.TempDir()); err == nil {
		t.Fatal("copied external symlink")
	}
}
