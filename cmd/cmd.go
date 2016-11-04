package cmd

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
	"jrubin.io/slog"
)

type Config struct {
	Logger               slog.Interface
	Cwd                  string
	GitCommit, BuildDate *string
}

type Constructor interface {
	New(app *cli.App, config *Config) cli.Command
}

func BashComplete(cmd cli.Command) cli.BashCompleteFunc {
	return func(c *cli.Context) {
		for _, command := range cmd.Subcommands {
			if command.Hidden {
				continue
			}

			for _, name := range command.Names() {
				fmt.Fprintln(c.App.Writer, name)
			}
		}

		for _, flag := range cmd.Flags {
			for _, name := range strings.Split(flag.GetName(), ",") {
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
}
