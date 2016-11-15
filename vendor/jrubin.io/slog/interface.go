package slog

import "io"

// PrefixWriteCloser is an io.WriteCloser that can be prefixed for every line
// it writes
type PrefixWriteCloser interface {
	io.WriteCloser
	Prefix(prefix string) PrefixWriteCloser
}

// Interface represents the API of both Logger and Entry.
type Interface interface {
	WithFields(fields Fielder) *Entry
	WithField(key string, value interface{}) *Entry
	WithError(err error) *Entry
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Fatal(msg string)
	Panic(msg string)
	IfError(error) Interface
	Trace(level Level, msg string) *Entry
	Writer(level Level) PrefixWriteCloser
}
