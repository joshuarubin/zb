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

type cc struct {
	zbcontext.Context
}

func (cmd *cc) New(_ *cli.App, ctx zbcontext.Context) cli.Command {
	cmd.Context = ctx
	cmd.ExcludeVendor = true

	return cli.Command{
		Name:      "commands",
		Usage:     "list all of the executables that will be emitted by the build command",
		ArgsUsage: "[packages]",
		Action: func(c *cli.Context) error {
			return cmd.run(c.App.Writer, c.Args()...)
		},
	}
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	if cmd.Package {
		return cmd.commandsPackage(w, args...)
	}

	return cmd.commandsProject(w, args...)
}

func (cmd *cc) commandsPackage(w io.Writer, args ...string) error {
	pkgs, err := project.ListPackages(cmd.Context, args...)
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

func (cmd *cc) commandsProject(w io.Writer, args ...string) error {
	projects, err := project.Projects(cmd.Context, args...)
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
