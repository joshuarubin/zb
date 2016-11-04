package list

import (
	"fmt"

	"github.com/urfave/cli"
	"jrubin.io/zb/buildflags"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/project"
)

var _ cmd.Constructor = (*List)(nil)

type List struct {
	Config     *cmd.Config
	BuildFlags buildflags.BuildFlags
	Vendor     bool
}

func (l *List) New(_ *cli.App, config *cmd.Config) cli.Command {
	l.Config = config

	cmd := cli.Command{
		Name:      "list",
		Usage:     "lists the packages in the repos of the packages named by the import paths, one per line.",
		ArgsUsage: "[-vendor] [build flags] [packages]",
		Action:    l.list,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "vendor",
				Usage:       "exclude vendor directories",
				Destination: &l.Vendor,
			},
		},
	}

	cmd.Flags = append(cmd.Flags, l.BuildFlags.Flags()...)

	return cmd
}

func (l *List) list(c *cli.Context) error {
	projects, err := project.Projects(
		l.BuildFlags.BuildContext(),
		l.Config.Cwd,
		l.Config.Logger,
		c.Args()...,
	)

	if err != nil {
		return err
	}

	pkgs, err := projects.Packages(l.Config.Logger)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		if l.Vendor && pkg.Vendor {
			continue
		}
		fmt.Fprintln(c.App.Writer, pkg.ImportPath)
	}

	return nil
}
