package build

import (
	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/cmd/install"
	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/dependency"
)

// Cmd is the build command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	buildflags.Data
}

func (co *cc) New(*cli.App) cli.Command {
	return cli.Command{
		Name:      "build",
		Usage:     "build all of the packages in each of the projects",
		ArgsUsage: "[build flags] [packages]",
		Action: func(c *cli.Context) error {
			ctx := cmd.Context(c)
			ctx.BuildContext = co.Data.BuildContext()
			ctx.BuildArger = co
			return install.Run(ctx, dependency.TargetBuild, c.Args()...)
		},
		Flags: co.BuildFlags(true),
	}
}
