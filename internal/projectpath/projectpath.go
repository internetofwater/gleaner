package projectpath

import (
	"path/filepath"
	"runtime"
)

// The gleaner config uses a relative path but we need to make sure that path is relative to the root
// at runtime so we can run tests with a relative path across the entire project
// Root allows us to get this info
var (
	_, b, _, _ = runtime.Caller(0)

	// Root folder of this project
	Root = filepath.Join(filepath.Dir(b), "../..")
)
