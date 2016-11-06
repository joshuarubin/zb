package commands

import (
	"fmt"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/project"
)

// Cmd is the commands command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	*cmd.Config
	BuildFlags buildflags.BuildFlags
	Context    *project.Context
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Config = config

	return cli.Command{
		Name:      "commands",
		Usage:     "list all of the executables that will be emitted by the build command",
		ArgsUsage: "[build flags] [packages]",
		Before:    cmd.setup,
		Action:    cmd.run,
		Flags:     cmd.BuildFlags.Flags(),
	}
}

func (cmd *cc) setup(_ *cli.Context) error {
	cmd.Context = &project.Context{
		BuildContext:  cmd.BuildFlags.BuildContext(),
		SrcDir:        cmd.Cwd,
		Logger:        cmd.Logger,
		ExcludeVendor: true,
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
			if exe := pkg.Command(); exe != nil {
				fmt.Fprintln(c.App.Writer, exe)
			}
		}
	}

	return nil
}
