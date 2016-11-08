package project

import (
	"go/build"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"jrubin.io/zb/lib/zbcontext"
)

// Projects lists the unique projects found by parsing the import paths in args
func Projects(ctx zbcontext.Context, args ...string) (ProjectList, error) {
	if len(args) == 0 {
		args = append(args, ".")
	}

	importPaths := ctx.ExpandEllipsis(args...)

	var projects ProjectList

	// don't use range, using importPaths as a queue
	for len(importPaths) > 0 {
		// pop the queue
		importPath := importPaths[0]
		importPaths = importPaths[1:]

		// convert local imports to import paths
		if build.IsLocalImport(importPath) {
			// convert relative path to absolute
			if !filepath.IsAbs(importPath) {
				importPath = filepath.Join(ctx.SrcDir, importPath)
			}

			if found := ctx.DirToImportPath(importPath); found != "" {
				importPath = found
			}
		}

		if dir := ctx.ImportPathToDir(importPath); dir != "" {
			if ok, _ := projects.Exists(dir); ok {
				continue
			}
		}

		p, err := project(ctx, importPath)

		if _, ok := err.(*build.NoGoError); ok && err != nil {
			// no buildable source files in the given dir
			// ok, as long as the project dir can still be found and at least
			// one subdir of the project dir has go files
			//
			// importPath may still be relative too, but it is guaranteed not to
			// have ellipsis

			newImportPaths := ctx.NoGoImportPathToProjectImportPaths(importPath)
			if len(newImportPaths) > 0 {
				// add the new paths to the queue and ignore the error
				importPaths = append(importPaths, newImportPaths...)
				continue
			}
		}

		if err != nil {
			return nil, err
		}

		if projects.Insert(p) {
			if err = p.fillPackages(); err != nil {
				return nil, err
			}
		}
	}

	return projects, nil
}

func project(ctx zbcontext.Context, importPath string) (*Project, error) {
	pkg, err := ctx.Import(importPath, ctx.SrcDir)
	if err != nil {
		return nil, err
	}

	pd := zbcontext.GitDir(pkg.Dir)
	if pd == "" {
		return nil, errors.Errorf("could not find project directory for: %s", pkg.Dir)
	}

	p := &Project{
		Context:  ctx,
		Dir:      pd,
		Packages: make([]*Package, 1),
	}

	p.Packages[0] = &Package{
		Package:    pkg,
		Project:    p,
		IsVendored: strings.Contains(pkg.Dir, "/vendor/"),
	}

	return p, nil
}
