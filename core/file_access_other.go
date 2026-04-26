//go:build !linux && !darwin

package core

import "os"

func ensureCoreLogDir(dir string, _ fileAccess) error {
	return os.MkdirAll(dir, 0o755)
}

func applyCoreLogFileAccess(_ string, _ fileAccess) error {
	return nil
}
