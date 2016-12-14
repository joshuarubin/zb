package clean

import (
	"io"
	"os"
	"path/filepath"

	"github.com/urfave/cli"
	"jrubin.io/slog"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the clean command
var Cmd cmd.Constructor = &cc{}

type cc struct{}

func (co *cc) New(_ *cli.App) cli.Command {
	return cli.Command{
		Name:      "clean",
		Usage:     "remove executables in repo produced by build",
		ArgsUsage: "[packages]",
		Action: func(c *cli.Context) error {
			ctx := cmd.Context(c)
			ctx.ExcludeVendor = true
			return co.run(ctx, c.App.Writer, c.Args()...)
		},
	}
}

func (co *cc) run(ctx zbcontext.Context, w io.Writer, args ...string) error {
	if ctx.Package {
		return co.cleanPackage(ctx, w, args...)
	}

	return co.cleanProject(ctx, w, args...)
}

func (co *cc) cleanPackage(ctx zbcontext.Context, w io.Writer, args ...string) error {
	pkgs, err := project.ListPackages(ctx, args...)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		cleanPackage(ctx.Logger, pkg.Dir, pkg)
	}

	return nil
}

func (co *cc) cleanProject(ctx zbcontext.Context, w io.Writer, args ...string) error {
	projects, err := project.Projects(ctx, args...)
	if err != nil {
		return err
	}

	for _, p := range projects {
		for _, pkg := range p.Packages {
			cleanPackage(ctx.Logger, p.Dir, pkg)
		}
	}

	return nil
}

func cleanPackage(logger slog.Interface, dir string, pkg *project.Package) {
	if !pkg.IsCommand() {
		return
	}

	path := pkg.BuildPath(dir)
	logger = logger.WithField("path", path)

	if rel, err := filepath.Rel(zbcontext.CWD, pkg.BuildPath(dir)); err == nil {
		logger = logger.WithField("path", rel)
	}

	err := os.Remove(path)

	if err == nil {
		logger.Info("removed")
		return
	}

	if os.IsNotExist(err) {
		logger.Info(err.(*os.PathError).Err.Error())
		return
	}

	logger.WithError(err).Error("error removing executable")
}
