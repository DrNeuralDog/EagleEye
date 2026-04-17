package platform

import (
	"os"
	"path/filepath"
	"strings"
)

var linuxExecutableSearchDirs = []string{
	"/usr/bin",
	"/bin",
	"/usr/local/bin",
}

func findAllowedExecutable(name string, searchDirs []string) (string, bool) {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return "", false
	}
	for _, dir := range searchDirs {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		return path, true
	}
	return "", false
}

// FindSystemExecutable returns a trusted system executable path for Linux helpers.
func FindSystemExecutable(name string) (string, bool) {
	return findAllowedExecutable(name, linuxExecutableSearchDirs)
}
