package cmd

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
	"jrubin.io/slog"
)

// Config is passed into each command's New Constructor
type Config struct {
	Logger               *slog.Logger
	Cwd                  string
	GitCommit, BuildDate *string
	NoWarnTodoFixme      bool
}

// Constructor returns a cli.Command
type Constructor interface {
	New(app *cli.App, config *Config) cli.Command
}

// BashComplete prints words suitable for completion of the App
func BashComplete(c *cli.Context) {
	bashComplete(c, c.App.Commands, c.App.Flags)
}

// BashCommandComplete prints words suitable for completion of a Command
func BashCommandComplete(cmd cli.Command) cli.BashCompleteFunc {
	return func(c *cli.Context) {
		bashComplete(c, cmd.Subcommands, cmd.Flags)
	}
}

func bashComplete(c *cli.Context, cmds []cli.Command, flags []cli.Flag) {
	for _, command := range cmds {
		if command.Hidden {
			continue
		}

		for _, name := range command.Names() {
			fmt.Fprintln(c.App.Writer, name)
		}
	}

	for _, flag := range flags {
		for _, name := range strings.Split(flag.GetName(), ",") {
			if name == cli.BashCompletionFlag.Name {
				continue
			}

			switch name = strings.TrimSpace(name); len(name) {
			case 0:
			case 1:
				fmt.Fprintln(c.App.Writer, "-"+name)
			default:
				fmt.Fprintln(c.App.Writer, "--"+name)
			}
		}
	}
}
