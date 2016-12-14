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
	buildflags.Data
}

func (co *cc) New(_ *cli.App) cli.Command {
	return cli.Command{
		Name:      "install",
		Usage:     "compile and install all of the packages in each of the projects",
		ArgsUsage: "[build flags] [packages]",
		Action: func(c *cli.Context) error {
			ctx := cmd.Context(c)
			ctx.BuildContext = co.Data.BuildContext()
			ctx.BuildArger = co
			return Run(ctx, dependency.TargetInstall, c.Args()...)
		},
		Flags: co.BuildFlags(true),
	}
}

func Run(ctx zbcontext.Context, tt dependency.TargetType, args ...string) error {
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

func installPackage(ctx zbcontext.Context, tt dependency.TargetType, args ...string) (int, error) {
	pkgs, err := project.ListPackages(ctx, args...)
	if err != nil {
		return 0, err
	}

	return pkgs.Build(ctx, tt)
}

func installProject(ctx zbcontext.Context, tt dependency.TargetType, args ...string) (int, error) {
	projects, err := project.Projects(ctx, args...)
	if err != nil {
		return 0, err
	}

	return projects.Build(ctx, tt)
}
