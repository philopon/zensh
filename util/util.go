package util

import (
	"os"
	"os/user"
	"path/filepath"
)

func ExpandPath(path string) (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}

	return SafeExpandPath(user.HomeDir, path), nil
}

func SafeExpandPath(home, path string) string {
	path = filepath.FromSlash(path)
	if len(path) >= 2 && path[0] == '~' && path[1] == os.PathSeparator {
		return filepath.Join(filepath.FromSlash(home), path[1:])
	}

	return path
}
