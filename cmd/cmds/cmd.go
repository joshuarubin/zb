package cmds

import (
	"fmt"

	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/project"
)

var _ cmd.Constructor = (*Cmd)(nil)

type Cmd struct {
	*cmd.Config
	BuildFlags buildflags.BuildFlags
	Projects   *project.Projects
}

func (cmd *Cmd) New(app *cli.App, config *cmd.Config) cli.Command {
	cmd.Config = config

	return cli.Command{
		Name:      "cmds",
		Usage:     "list all of the executables that will be emitted by the build command",
		ArgsUsage: "[build flags] [packages]",
		Before:    cmd.setup,
		Action:    cmd.run,
		Flags:     cmd.BuildFlags.Flags(),
	}
}

func (cmd *Cmd) setup(c *cli.Context) error {
	cmd.Projects = &project.Projects{
		BuildContext:  cmd.BuildFlags.BuildContext(),
		SrcDir:        cmd.Cwd,
		Logger:        cmd.Logger,
		ExcludeVendor: true,
	}

	return nil
}

func (cmd *Cmd) run(c *cli.Context) error {
	projects, err := cmd.Projects.List(c.Args()...)
	if err != nil {
		return err
	}

	for _, p := range projects {
		for _, pkg := range p.Packages {
			if exe := pkg.Cmd(); exe != "" {
				fmt.Fprintln(c.App.Writer, exe)
			}
		}
	}

	return nil
}
