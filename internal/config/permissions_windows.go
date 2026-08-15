//go:build windows

package config

// Windows does not use POSIX mode bits for access control. MkdirAll and
// WriteFile still receive the restrictive requested modes, but a redundant
// Chmod must not make an otherwise usable workspace fail to load.
func enforceDirectoryPermissions(path string) error {
	return nil
}

func enforceFilePermissions(path string) error {
	return nil
}
