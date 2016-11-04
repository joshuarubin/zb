package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"jrubin.io/slog"
	"jrubin.io/slog/handlers/text"
	"jrubin.io/zb/cmd"
	"jrubin.io/zb/cmd/list"
	"jrubin.io/zb/cmd/version"
	"jrubin.io/zb/project"
)

var (
	// populated by zb build ldflags
	GitCommit, BuildDate string

	logger slog.Logger

	level = slog.WarnLevel
	app   = cli.NewApp()

	config = cmd.Config{
		GitCommit: &GitCommit,
		BuildDate: &BuildDate,
		Logger:    &logger,
	}
)

var subcommands = []cmd.Constructor{
	&version.Cmd{},
	&list.Cmd{},
	// TODO(jrubin)
	// build
	// lint
	// test
	// imports? (list non-std, not-in-project recursive imports of project)
	// save? (copy imports to vendor/)
	// list out of date imports?
}

func gitCommit() (string, error) {
	// TODO(jrubin) delete this when set by zb build

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir, err := project.ProjectDir(cwd)
	if err != nil {
		return "", err
	}

	dir = filepath.Join(dir, ".git")

	repo, err := git.NewFilesystemRepository(dir)
	if err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", err
	}

	return head.Hash().String(), nil
}

const dateFormat = "2006-01-02T15:04:05+00:00"

func buildDate() string {
	// TODO(jrubin) delete this when set by zb build
	return time.Now().UTC().Format(dateFormat)
}

func init() {
	var err error
	GitCommit, err = gitCommit()
	if err != nil {
		panic(err)
	}

	BuildDate = buildDate()

	cli.ErrWriter = logger.Writer(slog.ErrorLevel)

	app.Name = "zb"
	app.HideVersion = true
	app.Version = "0.1.0"
	app.Usage = "an opinionated repo based build tool"
	app.EnableBashCompletion = true
	app.BashComplete = bashComplete
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
		if c.BashComplete == nil {
			c.BashComplete = cmd.BashComplete(c)
		}
		c.Before = wrapFn(c.Before)
		c.Action = wrapFn(c.Action)
		c.After = wrapFn(c.After)
		app.Commands = append(app.Commands, c)
	}
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func wrapFn(fn interface{}) func(*cli.Context) error {
	var do func(*cli.Context) error

	switch sig := fn.(type) {
	case func(*cli.Context) error:
		do = sig
	case cli.BeforeFunc:
		do = sig
	case cli.ActionFunc:
		do = sig
	case cli.AfterFunc:
		do = sig
	default:
		panic(errors.New("can't wrap invalid function signature"))
	}

	if do == nil {
		return nil
	}

	return func(c *cli.Context) error {
		err := do(c)
		if serr, ok := err.(stackTracer); ok && serr != nil {
			logger.
				WithError(err).
				WithField("command", c.Command.Name).
				Error("error")
			return errors.New("emitted stack trace")
		}
		return err
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

func bashComplete(c *cli.Context) {
	for _, command := range c.App.Commands {
		if command.Hidden {
			continue
		}

		for _, name := range command.Names() {
			fmt.Fprintln(c.App.Writer, name)
		}
	}

	for _, flag := range c.App.Flags {
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
