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
	buildflags.Data
	ExcludeVendor bool
}

func (co *cc) New(*cli.App) cli.Command {
	return cli.Command{
		Name:      "list",
		Usage:     "lists the packages in the repos of the packages named by the import paths, one per line.",
		ArgsUsage: "[-vendor] [build flags] [packages]",
		Action: func(c *cli.Context) error {
			ctx := cmd.Context(c)
			ctx.ExcludeVendor = co.ExcludeVendor
			ctx.BuildContext = co.Data.BuildContext()
			return co.run(ctx, c.App.Writer, c.Args()...)
		},
		Flags: append(co.BuildFlags(false),
			cli.BoolFlag{
				Name:        "vendor",
				Usage:       "exclude vendor directories",
				Destination: &co.ExcludeVendor,
			},
		),
	}
}

func (co *cc) run(ctx zbcontext.Context, w io.Writer, args ...string) error {
	if ctx.Package {
		return co.listPackage(ctx, w, args...)
	}

	return co.listProject(ctx, w, args...)
}

func (co *cc) listPackage(ctx zbcontext.Context, w io.Writer, args ...string) error {
	pkgs, err := project.ListPackages(ctx, args...)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		fmt.Fprintln(w, pkg.ImportPath)
	}

	return nil
}

func (co *cc) listProject(ctx zbcontext.Context, w io.Writer, args ...string) error {
	projects, err := project.Projects(ctx, args...)
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
