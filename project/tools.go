package project

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func ProjectDir(value string) (string, error) {
	dir := value
	for {
		test := filepath.Join(dir, ".git")
		_, err := os.Stat(test)
		if err != nil {
			ndir := filepath.Dir(dir)
			if ndir == dir {
				return "", errors.Errorf("could not find project dir for: %s", value)
			}

			dir = ndir
			continue
		}

		return dir, nil
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
	dir, err := ProjectDir(dir)
	if err != nil || dir == "" {
		return ""
	}
	return dir
}
