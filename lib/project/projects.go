package project

import (
	"go/build"

	"github.com/pkg/errors"

	"jrubin.io/zb/lib/zbcontext"
)

// Projects lists the unique projects found by parsing the import paths in args
func Projects(ctx zbcontext.Context, args ...string) (List, error) {
	if len(args) == 0 {
		args = append(args, ".")
	}

	importPaths := ctx.ExpandEllipsis(args...)

	var projects List

	// don't use range, using importPaths as a queue
	for len(importPaths) > 0 {
		// pop the queue
		importPath := ctx.NormalizeImportPath(importPaths[0])
		importPaths = importPaths[1:]

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
			if err = p.fillPackages(ctx); err != nil {
				return nil, err
			}
		}
	}

	return projects, nil
}

func project(ctx zbcontext.Context, importPath string) (*Project, error) {
	p := &Project{
		Packages: make([]*Package, 1),
	}

	pkg, err := NewPackage(ctx, importPath, zbcontext.CWD, true)
	if err != nil {
		return nil, err
	}

	p.Dir = zbcontext.GitDir(pkg.Package.Dir)
	if p.Dir == "" {
		return nil, errors.Errorf("could not find project directory for: %s", pkg.Package.Dir)
	}

	p.Packages[0] = pkg

	return p, nil
}
