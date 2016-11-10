package main

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

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
			config.Logger.
				WithError(err).
				WithField("command", c.Command.Name).
				Error("error")
			return errors.New("emitted stack trace")
		}
		return err
	}
}
