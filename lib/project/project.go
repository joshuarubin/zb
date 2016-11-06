package project

import (
	"go/build"
	"path/filepath"
	"strings"

	"jrubin.io/slog"
	"jrubin.io/zb/lib/ellipsis"

	"github.com/pkg/errors"
)

// A Project is a collection of Packages contained within a single repository
type Project struct {
	Dir           string
	Packages      []*Package
	BuildContext  build.Context
	ExcludeVendor bool
	Logger        slog.Interface

	filled bool
}

func (p *Project) fillPackages() error {
	if p.filled {
		return nil
	}

	p.filled = true

	base := dirToImportPath(p.BuildContext, p.Dir)
	if base == "" {
		return errors.Errorf("could not find base import path for: %s", p.Dir)
	}

	// base should always be a fully qualified package import, never an absolute
	// or relative path

	list := (*packageList)(&p.Packages)

	importPaths := ellipsis.Expand(p.BuildContext, p.Logger, filepath.Join(base, "..."))
	for _, importPath := range importPaths {
		if dir := importPathToDir(p.BuildContext, importPath); dir != "" {
			if ok, _ := list.Exists(dir); ok {
				continue
			}
		}

		// TODO(jrubin) this isn't a very accurate check
		isVendored := strings.Contains(importPath, "/vendor/")

		if p.ExcludeVendor && isVendored {
			continue
		}

		pkg, err := p.BuildContext.Import(importPath, "", build.ImportComment)
		if err != nil {
			return err
		}

		list.Insert(&Package{
			Package:      pkg,
			Project:      p,
			IsVendored:   isVendored,
			Logger:       p.Logger,
			BuildContext: p.BuildContext,
		})
	}

	return nil
}

func (p *Project) Targets() ([]*Target, error) {
	var targets []*Target
	for _, pkg := range p.Packages {
		t, err := pkg.Targets()
		if err != nil {
			return nil, err
		}
		targets = append(targets, t...)
	}
	return targets, nil
}
