package project

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

// Dir checks the directory value for the presence of .git and will walk up the
// filesystem hierarchy to find one. Returns an empty string if no directory
// containing .git was found.
func Dir(value string) string {
	dir := value
	for {
		test := filepath.Join(dir, ".git")
		_, err := os.Stat(test)
		if err != nil {
			ndir := filepath.Dir(dir)
			if ndir == dir {
				return ""
			}

			dir = ndir
			continue
		}

		return dir
	}
}

// can't handle ellipsis (...), but does not require .go files to exist either
func importPathToDir(bc build.Context, importPath string) string {
	for _, srcDir := range bc.SrcDirs() {
		dir := filepath.Join(srcDir, importPath)
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		return dir
	}
	return ""
}

func dirToImportPath(bc build.Context, dir string) string {
	// path may be a/b/c/d
	// p.Dir may be /home/user/go/src/a/b
	// this will return a/b even if there are no .go files in it
	// e.g. it may not be a valid import path

	for _, srcDir := range bc.SrcDirs() {
		srcDir += "/"
		if strings.Index(dir, srcDir) == 0 {
			return dir[len(srcDir):]
		}
	}
	return ""
}

func importPathToProjectDir(bc build.Context, importPath string) string {
	dir := importPathToDir(bc, importPath)
	if dir == "" {
		return ""
	}
	return Dir(dir)
}
