package complete

import (
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"jrubin.io/zb/cmd"
)

// Cmd is the complete command
var Cmd cmd.Constructor = &cc{}

type shell int

const (
	bash shell = iota
	zsh
)

type cc struct {
	AppName  string
	FlagName string
	Shell    shell
	Bash     shell
	Zsh      shell
}

func (co *cc) New(app *cli.App) cli.Command {
	co.Bash = bash
	co.Zsh = zsh
	co.AppName = app.Name
	co.FlagName = cli.BashCompletionFlag.Name

	return cli.Command{
		Name:        "complete",
		Usage:       "generate autocomplete script",
		Description: `eval "$(zb complete)"`,
		Before:      co.setup,
		Action:      co.run,
	}
}

func (co *cc) setup(c *cli.Context) error {
	switch shell := filepath.Base(os.Getenv("SHELL")); shell {
	case "bash":
		co.Shell = bash
	case "zsh":
		co.Shell = zsh
	default:
		return errors.Errorf("unsupported shell: %s", shell)
	}
	return nil
}

func (co *cc) run(c *cli.Context) error {
	return tpl.Execute(c.App.Writer, co)
}

var tpl *template.Template

func init() {
	tpl = template.Must(template.New("shellFunc").Parse(shellFunc))
}

var shellFunc = `{{ if eq .Shell .Bash }}#!/bin/bash{{ end }}{{ if eq .Shell .Zsh }}autoload -U compinit && compinit
autoload -U bashcompinit && bashcompinit{{ end }}

_{{ .AppName }}_autocomplete() {
     local cur opts base
     COMPREPLY=()
     cur="${COMP_WORDS[COMP_CWORD]}"
     opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --{{ .FlagName }} )
     COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
     return 0
 }

 complete -o default -F _{{ .AppName }}_autocomplete {{ .AppName }}
`
