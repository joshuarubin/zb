package install

import (
	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/dependency"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the install command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zbcontext.Context
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Config = config

	return cli.Command{
		Name:      "install",
		Usage:     "compile and install all of the packages in each of the projects",
		ArgsUsage: "[build flags] [packages]",
		Action: func(c *cli.Context) error {
			return Run(&cmd.Context, dependency.TargetInstall, c.Args()...)
		},
		Flags: append(cmd.BuildFlags(),
			cli.StringFlag{
				Name: "run",
				Usage: `

				passed to "go generate" if non-empty, specifies a regular
				expression to select directives whose full original source text
				(excluding any trailing spaces and final newline) matches the
				expression.`,
				Destination: &cmd.GenerateRun,
			},
		),
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
		ctx.Logger.Info("nothing to install")
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
