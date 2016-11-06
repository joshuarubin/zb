package main

import (
	"os"

	"github.com/urfave/cli"

	"jrubin.io/slog"
	"jrubin.io/slog/handlers/text"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/cmd/build"
	"jrubin.io/zb/cmd/commands"
	"jrubin.io/zb/cmd/complete"
	"jrubin.io/zb/cmd/list"
	"jrubin.io/zb/cmd/version"
)

var (
	// populated by zb build ldflags
	gitCommit, buildDate string

	logger slog.Logger

	level = slog.WarnLevel
	app   = cli.NewApp()

	config = cmd.Config{
		GitCommit: &gitCommit,
		BuildDate: &buildDate,
		Logger:    &logger,
	}
)

var subcommands = []cmd.Constructor{
	version.Cmd,
	list.Cmd,
	commands.Cmd,
	complete.Cmd,
	build.Cmd,
	// TODO(jrubin)
	// install
	// lint
	// test (with cache like gt)
	// imports? (list non-std, not-in-project recursive imports of project)
	// save? (copy imports to vendor/)
	// list out of date vendor imports?
	// build (others?) should show TODO/FIXME comments
}

func init() {
	app.Name = "zb"
	app.HideVersion = true
	app.Version = "0.1.0"
	app.Usage = "an opinionated repo based build tool"
	app.EnableBashCompletion = true
	app.BashComplete = cmd.BashComplete
	app.Before = setup
	app.Authors = []cli.Author{
		{Name: "Joshua Rubin", Email: "joshua@rubixconsulting.com"},
	}
	app.Flags = []cli.Flag{
		cli.GenericFlag{
			Name:   "log-level",
			EnvVar: "LOG_LEVEL",
			Usage:  "set log level (DEBUG, INFO, WARN, ERROR)",
			Value:  &level,
		},
	}

	for _, sc := range subcommands {
		c := sc.New(app, &config)
		c.Before = wrapFn(c.Before)
		c.Action = wrapFn(c.Action)
		c.After = wrapFn(c.After)
		if c.BashComplete == nil {
			c.BashComplete = cmd.BashCommandComplete(c)
		}
		app.Commands = append(app.Commands, c)
	}
}

func main() {
	_ = app.Run(os.Args) // nosec
}

func setup(c *cli.Context) error {
	var err error

	logger.RegisterHandler(level, &text.Handler{
		Writer:           os.Stderr,
		DisableTimestamp: true,
	})

	config.Cwd, err = os.Getwd()
	return err
}
