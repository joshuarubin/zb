package project

import (
	"go/build"
	"path/filepath"

	"github.com/pkg/errors"

	"jrubin.io/slog"
	"jrubin.io/zb/lib/ellipsis"
)

// Context for package related commands
type Context struct {
	BuildContext  build.Context
	BuildFlags    []string
	SrcDir        string
	ExcludeVendor bool
	Logger        *slog.Logger
}

// Projects lists the unique projects found by parsing the import paths in args
func (ctx *Context) Projects(args ...string) (Projects, error) {
	if len(args) == 0 {
		args = append(args, ".")
	}

	importPaths := ellipsis.Expand(ctx.BuildContext, ctx.Logger, args...)

	var projects Projects

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

			if found := dirToImportPath(ctx.BuildContext, importPath); found != "" {
				importPath = found
			}
		}

		if dir := importPathToDir(ctx.BuildContext, importPath); dir != "" {
			if ok, _ := projects.Exists(dir); ok {
				continue
			}
		}

		p, err := ctx.project(importPath)

		if _, ok := err.(*build.NoGoError); ok && err != nil {
			// no buildable source files in the given dir
			// ok, as long as the project dir can still be found and at least
			// one subdir of the project dir has go files
			//
			// importPath may still be relative too, but it is guaranteed not to
			// have ellipsis

			newImportPaths := ctx.noGoImportPathToProjectImportPaths(importPath)
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

func (ctx *Context) project(importPath string) (*Project, error) {
	pkg, err := ctx.BuildContext.Import(importPath, ctx.SrcDir, build.ImportComment)
	if err != nil {
		return nil, err
	}

	pd := Dir(pkg.Dir)
	if pd == "" {
		return nil, errors.Errorf("could not find project directory for: %s", pkg.Dir)
	}

	p := &Project{
		Dir:           pd,
		BuildContext:  ctx.BuildContext,
		BuildFlags:    ctx.BuildFlags,
		Packages:      make([]*Package, 1),
		Logger:        ctx.Logger,
		ExcludeVendor: ctx.ExcludeVendor,
	}

	p.Packages[0] = &Package{
		Package:      pkg,
		Project:      p,
		Logger:       ctx.Logger,
		IsVendored:   false, // TODO(jrubin)
		BuildContext: ctx.BuildContext,
		BuildFlags:   ctx.BuildFlags,
	}

	return p, nil
}

func (ctx *Context) noGoImportPathToProjectImportPaths(importPath string) []string {
	dir := importPathToProjectDir(ctx.BuildContext, importPath)
	if dir == "" {
		return nil
	}

	// found project dir, now convert it back to an import path so
	// we can use ellipsis
	importPath = dirToImportPath(ctx.BuildContext, dir)

	// add the ellipsis
	importPath = filepath.Join(importPath, "...")

	// lets see if we can find any packages under it
	return ellipsis.Expand(ctx.BuildContext, ctx.Logger, importPath)
}
