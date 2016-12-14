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

type cc struct {
	zbcontext.Context
}

func (cmd *cc) New(_ *cli.App, ctx zbcontext.Context) cli.Command {
	cmd.Context = ctx
	cmd.ExcludeVendor = true

	return cli.Command{
		Name:      "clean",
		Usage:     "remove executables in repo produced by build",
		ArgsUsage: "[packages]",
		Action: func(c *cli.Context) error {
			return cmd.run(c.App.Writer, c.Args()...)
		},
	}
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	if cmd.Package {
		return cmd.cleanPackage(w, args...)
	}

	return cmd.cleanProject(w, args...)
}

func (cmd *cc) cleanPackage(w io.Writer, args ...string) error {
	pkgs, err := project.ListPackages(cmd.Context, args...)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		cleanPackage(&cmd.Logger, pkg.Dir, pkg)
	}

	return nil
}

func (cmd *cc) cleanProject(w io.Writer, args ...string) error {
	projects, err := project.Projects(cmd.Context, args...)
	if err != nil {
		return err
	}

	for _, p := range projects {
		for _, pkg := range p.Packages {
			cleanPackage(&cmd.Logger, p.Dir, pkg)
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
