package install

import (
	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/buildflags"
	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the install command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	*zbcontext.Context
	buildflags.Data
}

func (cmd *cc) New(_ *cli.App, ctx *zbcontext.Context) cli.Command {
	cmd.Context = ctx

	return cli.Command{
		Name:      "install",
		Usage:     "compile and install all of the packages in each of the projects",
		ArgsUsage: "[build flags] [packages]",
		Action: func(c *cli.Context) error {
			cmd.Context.BuildContext = cmd.Data.BuildContext()
			cmd.Context.BuildArger = cmd
			return Run(cmd.Context, dependency.TargetInstall, c.Args()...)
		},
		Flags: cmd.BuildFlags(true),
	}
}

func Run(ctx *zbcontext.Context, tt dependency.TargetType, args ...string) error {
	var err error
	var built int

	if ctx.Package {
		built, err = installPackage(ctx, tt, args...)
	} else {
		built, err = installProject(ctx, tt, args...)
	}

	if err != nil {
		return err
	}

	if built == 0 {
		ctx.Logger.Info("nothing to " + tt.String())
	}

	return nil
}

func installPackage(ctx *zbcontext.Context, tt dependency.TargetType, args ...string) (int, error) {
	pkgs, err := project.ListPackages(ctx, args...)
	if err != nil {
		return 0, err
	}

	return pkgs.Build(tt)
}

func installProject(ctx *zbcontext.Context, tt dependency.TargetType, args ...string) (int, error) {
	projects, err := project.Projects(ctx, args...)
	if err != nil {
		return 0, err
	}

	return projects.Build(tt)
}
