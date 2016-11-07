package build

import (
	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/lib/project"
	"jrubin.io/zb/lib/zbcontext"
)

// TODO(jrubin) automatically add missing imports to vendor/

// Cmd is the build command
var Cmd cmd.Constructor = &cc{}

type cc struct {
	zbcontext.Context
}

func (cmd *cc) New(_ *cli.App, config *cmd.Config) cli.Command {
	cmd.Logger = config.Logger
	cmd.SrcDir = config.Cwd

	return cli.Command{
		Name:      "build",
		Usage:     "build all of the executables in each of the projects",
		ArgsUsage: "[build flags] [packages]",
		Action: func(c *cli.Context) error {
			return cmd.run(c.Args()...)
		},
		Flags: append(cmd.Flags(),
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
	projects, err := project.Projects(cmd.Context, args...)
	if err != nil {
		return err
	}

	targets, err := projects.Targets(project.TargetBuild)
	if err != nil {
		return err
	}

	var built bool

	for _, target := range targets {
		if !target.Buildable() {
			continue
		}

		deps, err := target.Dependencies()
		if err != nil {
			return err
		}

		// build target if any of its dependencies are newer than itself

		for _, d := range deps {
			if d.ModTime().After(target.ModTime()) {
				// TODO(jrubin) run builds in parallel where possible

				if err := target.Build(); err != nil {
					return err
				}

				built = true
				break
			}
		}
	}

	if !built {
		cmd.Logger.Info("nothing to build")
	}

	return nil
}
