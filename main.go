package main

import (
	"os"

	"github.com/urfave/cli"

	"jrubin.io/slog"
	"jrubin.io/slog/handlers/text"
	"jrubin.io/zb/buildflags"
	"jrubin.io/zb/pkgs"
	"jrubin.io/zb/project"
)

const version = "0.1.0"

var (
	bf     buildflags.BuildFlags
	logger slog.Logger
	level  = slog.WarnLevel
	app    = cli.NewApp()
)

func init() {
	cli.ErrWriter = logger.Writer(slog.ErrorLevel)

	app.Name = "zb"
	app.HideVersion = true
	app.Version = version
	app.Usage = "repo based build tool"
	app.Before = setup
	app.Action = run
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
	app.Flags = append(app.Flags, bf.Flags()...)
}

func main() {
	_ = app.Run(os.Args) // nosec
}

func setup(c *cli.Context) error {
	logger.RegisterHandler(level, text.New(os.Stderr))
	return nil
}

// TODO(jrubin) build [-i] flag

func run(c *cli.Context) error {
	// resolve the package paths given on the command line
	args := c.Args()
	if len(args) == 0 {
		args = append(args, "./...")
	}

	bc := bf.BuildContext()

	paths := pkgs.ImportPaths(bc, args...)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	projects := map[string]*project.Project{}
	for _, path := range paths {
		p, err := project.New(&logger, bc, path, cwd)
		if err != nil {
			return err
		}

		if _, ok := projects[p.Dir]; ok {
			continue
		}

		projects[p.Dir] = p
	}

	return nil
}
