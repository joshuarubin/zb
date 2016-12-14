package build

import (
	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/cmd/install"
	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the build command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zbcontext.Context
	buildflags.Data
}

func (cmd *cc) New(_ *cli.App, ctx zbcontext.Context) cli.Command {
	cmd.Context = ctx

	return cli.Command{
		Name:      "build",
		Usage:     "build all of the packages in each of the projects",
		ArgsUsage: "[build flags] [packages]",
		Action: func(c *cli.Context) error {
			cmd.Context.BuildContext = cmd.Data.BuildContext()
			cmd.Context.BuildArger = cmd
			return install.Run(cmd.Context, dependency.TargetBuild, c.Args()...)
		},
		Flags: cmd.BuildFlags(true),
	}
}
