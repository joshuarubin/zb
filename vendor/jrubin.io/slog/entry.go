package slog

import (
	"errors"
	"os"
	"time"
)

// assert interface compliance.
var _ Interface = (*Entry)(nil)

// Entry represents a single log entry.
type Entry struct {
	Logger     *Logger   `json:"-"`
	Fields     Fields    `json:"fields"`
	Level      Level     `json:"level"`
	Time       time.Time `json:"time"`
	Message    string    `json:"msg"`
	start      time.Time
	fields     []Fields
	traceLevel Level
}

// NewEntry returns a new entry for `log`.
func NewEntry(log *Logger) *Entry {
	return &Entry{
		Logger: log,
	}
}

// WithFields returns a new entry with `fields` set.
func (e *Entry) WithFields(fields Fielder) *Entry {
	return &Entry{
		Logger: e.Logger,
		fields: append(e.fields, fields.Fields()),
	}
}

// WithField returns a new entry with the `key` and `value` set.
func (e *Entry) WithField(key string, value interface{}) *Entry {
	return e.WithFields(Fields{key: value})
}

// WithError returns a new entry with the "error" set to `err`.
func (e *Entry) WithError(err error) *Entry {
	return e.WithField("error", err)
}

// Debug level message.
func (e *Entry) Debug(msg string) {
	e.Logger.log(DebugLevel, e, msg)
}

// Info level message.
func (e *Entry) Info(msg string) {
	e.Logger.log(InfoLevel, e, msg)
}

// Warn level message.
func (e *Entry) Warn(msg string) {
	e.Logger.log(WarnLevel, e, msg)
}

// Error level message.
func (e *Entry) Error(msg string) {
	e.Logger.log(ErrorLevel, e, msg)
}

// Fatal level message, followed by an exit.
func (e *Entry) Fatal(msg string) {
	if e.Logger != Nil {
		e.Logger.log(FatalLevel, e, msg)
		os.Exit(1)
	}
}

// Panic level message, followed by a panic.
func (e *Entry) Panic(msg string) {
	if e.Logger != Nil {
		e.Logger.log(PanicLevel, e, msg)
		panic(errors.New(msg))
	}
}

// IfError returns an Interface that will only log if err is not nil
func (e *Entry) IfError(err error) Interface {
	if err != nil {
		return e.WithError(err)
	}

	return Nil
}

// Trace returns a new entry with a Stop method to fire off
// a corresponding completion log, useful with defer.
func (e *Entry) Trace(level Level, msg string) *Entry {
	e.Logger.log(level, e, msg)
	v := e.WithFields(e.Fields)
	v.Message = msg
	v.start = time.Now()
	v.traceLevel = level
	return v
}

// Stop should be used with Trace, to fire off the completion message. When
// an `err` is passed the "error" field is set, and the log level is error.
func (e *Entry) Stop(err *error) {
	v := e.WithField("duration", time.Since(e.start))

	if err == nil || *err == nil {
		v.Logger.log(e.traceLevel, v, e.Message)
		return
	}

	v.WithError(*err).Error(e.Message)
}

// mergedFields returns the fields list collapsed into a single map.
func (e *Entry) mergedFields() Fields {
	f := Fields{}

	for _, fields := range e.fields {
		for k, v := range fields {
			f[k] = v
		}
	}

	return f
}

// finalize returns a copy of the Entry with Fields merged.
func (e *Entry) finalize(level Level, msg string) *Entry {
	return &Entry{
		Logger:  e.Logger,
		Fields:  e.mergedFields(),
		Level:   level,
		Message: msg,
		Time:    time.Now(),
	}
}
