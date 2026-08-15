//go:build !windows

package config

import "os"

func enforceDirectoryPermissions(path string) error {
	return os.Chmod(path, dirPermission)
}

func enforceFilePermissions(path string) error {
	return os.Chmod(path, filePermission)
}
