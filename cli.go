package main

import (
	"github.com/urfave/cli"
	"jrubin.io/slog"
)

func init() {
	cli.ErrWriter = logger.Writer(slog.ErrorLevel)

	cli.BashCompletionFlag = cli.BoolFlag{
		Name:   "compgen",
		Hidden: true,
	}

	cli.CommandHelpTemplate = `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{if .Category}}

CATEGORY:
   {{.Category}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .VisibleFlags}}

OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`
}
