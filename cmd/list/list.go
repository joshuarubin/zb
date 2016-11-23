package list

import (
	"fmt"
	"io"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the list command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	*zbcontext.Context
	buildflags.Data
}

func (cmd *cc) New(_ *cli.App, ctx *zbcontext.Context) cli.Command {
	cmd.Context = ctx

	return cli.Command{
		Name:      "list",
		Usage:     "lists the packages in the repos of the packages named by the import paths, one per line.",
		ArgsUsage: "[-vendor] [build flags] [packages]",
		Action: func(c *cli.Context) error {
			return cmd.run(c.App.Writer, c.Args()...)
		},
		Flags: append(cmd.BuildFlags(false),
			cli.BoolFlag{
				Name:        "vendor",
				Usage:       "exclude vendor directories",
				Destination: &cmd.ExcludeVendor,
			},
		),
	}
}

func (cmd *cc) run(w io.Writer, args ...string) error {
	cmd.Context.BuildContext = cmd.Data.BuildContext()

	if cmd.Package {
		return cmd.listPackage(w, args...)
	}

	return cmd.listProject(w, args...)
}

func (cmd *cc) listPackage(w io.Writer, args ...string) error {
	pkgs, err := project.ListPackages(cmd.Context, args...)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		fmt.Fprintln(w, pkg.ImportPath)
	}

	return nil
}

func (cmd *cc) listProject(w io.Writer, args ...string) error {
	projects, err := project.Projects(cmd.Context, args...)
	if err != nil {
		return err
	}

	for _, p := range projects {
		for _, pkg := range p.Packages {
			fmt.Fprintln(w, pkg.ImportPath)
		}
	}

	return nil
}
