package collector

import (
	"path/filepath"
	"time"
)

// This file is used and compiled only for tests, and allows us to
// edit the unexported variables with the values matching the test environment.
var TestFS = func(path string) {
	rootPrefix = path
	sysfsDir = filepath.Join(rootPrefix, "sys")
	accelFSRoot = filepath.Join(sysfsDir, "class/accel")
}

var Now = func(tf func() time.Time) {
	now = tf
}
