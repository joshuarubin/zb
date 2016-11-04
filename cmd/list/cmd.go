package list

import (
	"fmt"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/project"
)

// Cmd is the list command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	*cmd.Config
	BuildFlags    buildflags.BuildFlags
	ExcludeVendor bool
	Context       *project.Context
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Config = config

	ret := cli.Command{
		Name:      "list",
		Usage:     "lists the packages in the repos of the packages named by the import paths, one per line.",
		ArgsUsage: "[-vendor] [build flags] [packages]",
		Before:    cmd.setup,
		Action:    cmd.run,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:        "vendor",
				Usage:       "exclude vendor directories",
				Destination: &cmd.ExcludeVendor,
			},
		},
	}

	ret.Flags = append(ret.Flags, cmd.BuildFlags.Flags()...)

	return ret
}

func (cmd *cc) setup(c *cli.Context) error {
	cmd.Context = &project.Context{
		BuildContext:  cmd.BuildFlags.BuildContext(),
		SrcDir:        cmd.Cwd,
		Logger:        cmd.Logger,
		ExcludeVendor: cmd.ExcludeVendor,
	}

	return nil
}

func (cmd *cc) run(c *cli.Context) error {
	projects, err := cmd.Context.Projects(c.Args()...)
	if err != nil {
		return err
	}

	for _, p := range projects {
		for _, pkg := range p.Packages {
			fmt.Fprintln(c.App.Writer, pkg.ImportPath)
		}
	}

	return nil
}
