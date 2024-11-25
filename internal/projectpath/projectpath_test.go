package projectpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(t *testing.T) {
	// Check that main.go exists at the root
	mainFile := filepath.Join(Root, "main.go")
	// check that projectpath.Root/main.go exists
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Error("main.go not found at projectpath.Root", err)
	}
}
