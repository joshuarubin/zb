package project

import (
	"go/build"
	"path/filepath"

	"jrubin.io/slog"
	"jrubin.io/zb/lib/ellipsis"
)

type Projects struct {
	BuildContext  build.Context
	SrcDir        string
	ExcludeVendor bool
	Logger        slog.Interface
}

func (ps *Projects) List(args ...string) ([]*Project, error) {
	if len(args) == 0 {
		args = append(args, ".")
	}

	importPaths := ellipsis.Expand(ps.BuildContext, ps.Logger, args...)

	var projects projectList

	// don't use range, using importPaths as a queue
	for len(importPaths) > 0 {
		// pop the queue
		importPath := importPaths[0]
		importPaths = importPaths[1:]

		// convert local imports to import paths
		if build.IsLocalImport(importPath) {
			// convert relative path to absolute
			if !filepath.IsAbs(importPath) {
				importPath = filepath.Join(ps.SrcDir, importPath)
			}

			if found := dirToImportPath(ps.BuildContext, importPath); found != "" {
				importPath = found
			}
		}

		if dir := importPathToDir(ps.BuildContext, importPath); dir != "" {
			if ok, _ := projects.Exists(dir); ok {
				continue
			}
		}

		p, err := ps.Project(importPath)

		if _, ok := err.(*build.NoGoError); ok && err != nil {
			// no buildable source files in the given dir
			// ok, as long as the project dir can still be found and at least
			// one subdir of the project dir has go files
			//
			// importPath may still be relative too, but it is guaranteed not to
			// have ellipsis

			newImportPaths := ps.noGoImportPathToProjectImportPaths(importPath)
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

func (ps *Projects) Project(importPath string) (*Project, error) {
	pkg, err := build.Import(importPath, ps.SrcDir, build.ImportComment)
	if err != nil {
		return nil, err
	}

	pd, err := ProjectDir(pkg.Dir)
	if err != nil {
		return nil, err
	}

	p := &Project{
		Dir:           pd,
		BuildContext:  ps.BuildContext,
		Packages:      make([]*Package, 1),
		Logger:        ps.Logger,
		ExcludeVendor: ps.ExcludeVendor,
	}

	p.Packages[0] = &Package{
		Package:    pkg,
		Project:    p,
		IsVendored: false, // TODO(jrubin)
	}

	return p, nil
}

func (ps *Projects) noGoImportPathToProjectImportPaths(importPath string) []string {
	dir := importPathToProjectDir(ps.BuildContext, importPath)
	if dir == "" {
		return nil
	}

	// found project dir, now convert it back to an import path so
	// we can use ellipsis
	importPath = dirToImportPath(ps.BuildContext, dir)

	// add the ellipsis
	importPath = filepath.Join(importPath, "...")

	// lets see if we can find any packages under it
	return ellipsis.Expand(ps.BuildContext, ps.Logger, importPath)
}
