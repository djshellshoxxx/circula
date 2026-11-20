package messaging

import "errors"

var errPrivateStorage = errors.New("messaging database must be a private regular file")

// CreatePrivateDirectory explicitly provisions a new standalone storage
// directory. It never changes an existing directory. On Windows the directory
// has a protected process-user-only DACL inherited by SQLite and backup files;
// its process default owner must also be the process user. See CONTRACT.md.
// On other systems it creates a directory with mode 0700.
func CreatePrivateDirectory(path string) error { return createPrivateDirectory(path) }
