package commands

import (
	"fmt"
	"io"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the commands command
var Cmd cmd.Constructor = &cc{}

type cc struct{}

func (co *cc) New(_ *cli.App) cli.Command {
	return cli.Command{
		Name:      "commands",
		Usage:     "list all of the executables that will be emitted by the build command",
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
		return co.commandsPackage(ctx, w, args...)
	}

	return co.commandsProject(ctx, w, args...)
}

func (co *cc) commandsPackage(ctx zbcontext.Context, w io.Writer, args ...string) error {
	pkgs, err := project.ListPackages(ctx, args...)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		if pkg.IsCommand() {
			fmt.Fprintln(w, pkg.BuildPath(pkg.Dir))
		}
	}

	return nil
}

func (co *cc) commandsProject(ctx zbcontext.Context, w io.Writer, args ...string) error {
	projects, err := project.Projects(ctx, args...)
	if err != nil {
		return err
	}

	for _, p := range projects {
		for _, pkg := range p.Packages {
			if pkg.IsCommand() {
				fmt.Fprintln(w, pkg.BuildPath(p.Dir))
			}
		}
	}

	return nil
}
