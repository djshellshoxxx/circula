//go:build !windows

package messaging

import (
	"io"
	"os"
)

func createPrivateDirectory(path string) error { return os.Mkdir(path, 0700) }

func preparePrivateDatabase(path string) (string, io.Closer, error) {
	// Preserve the existing non-Windows creation and permission checks.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return "", nil, err
	}
	info, err := file.Stat()
	closeErr := file.Close()
	if err != nil {
		return "", nil, err
	}
	if closeErr != nil {
		return "", nil, closeErr
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return "", nil, errPrivateStorage
	}
	return path, nil, nil
}

func privateDatabaseDSN(path string) string { return path }
