package build

import (
	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// Cmd is the build command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zbcontext.Context
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Logger = config.Logger
	cmd.SrcDir = config.Cwd
	cmd.NoWarnTodoFixme = &config.NoWarnTodoFixme

	return cli.Command{
		Name:      "build",
		Usage:     "build all of the packages in each of the projects",
		ArgsUsage: "[build flags] [packages]",
		Action: func(c *cli.Context) error {
			return cmd.run(c.Args()...)
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

func (cmd *cc) run(args ...string) error {
	projects, err := project.Projects(&cmd.Context, args...)
	if err != nil {
		return err
	}

	built, err := projects.Build(project.TargetBuild)
	if err != nil {
		return err
	}

	if built == 0 {
		cmd.Logger.Info("nothing to build")
	}

	return nil
}
