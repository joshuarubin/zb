package slog

import "log"

// assert interface compliance.
var _ Interface = (*Logger)(nil)

// Fielder is an interface for providing fields to custom types.
type Fielder interface {
	Fields() Fields
}

// Fields represents a map of entry level data used for structured logging.
type Fields map[string]interface{}

// Fields implements Fielder.
func (f Fields) Fields() Fields {
	return f
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// log handlers. If f is a function with the appropriate signature,
// HandlerFunc(f) is a Handler object that calls f.
type HandlerFunc func(*Entry) error

// HandleLog calls f(e).
func (f HandlerFunc) HandleLog(e *Entry) error {
	return f(e)
}

// Handler is used to handle log events, outputting them to
// stdio or sending them to remote services. See the "handlers"
// directory for implementations.
//
// It is left up to Handlers to implement thread-safety.
type Handler interface {
	HandleLog(*Entry) error
}

// Logger represents a logger. It will not do anything useful unless a handler
// has been registered with RegisterHandler.
type Logger struct {
	handlers [len(levelNames)][]Handler
}

// New allocates a new Logger.
func New() *Logger {
	return &Logger{}
}

// RegisterHandler adds a new Handler and specifies the maximum Level that the
// handler will be passed log entries for
func (l *Logger) RegisterHandler(maxLevel Level, handler Handler) *Logger {
	if maxLevel < PanicLevel {
		maxLevel = PanicLevel
	}

	if maxLevel > DebugLevel {
		maxLevel = DebugLevel
	}

	for level := PanicLevel; level <= maxLevel; level++ {
		if handlers := l.handlers[level]; handlers != nil {
			l.handlers[level] = append(handlers, handler)
			continue
		}

		l.handlers[level] = []Handler{handler}
	}

	return l
}

// WithFields returns a new entry with `fields` set.
func (l *Logger) WithFields(fields Fielder) *Entry {
	return NewEntry(l).WithFields(fields.Fields())
}

// WithField returns a new entry with the `key` and `value` set.
func (l *Logger) WithField(key string, value interface{}) *Entry {
	return NewEntry(l).WithField(key, value)
}

// WithError returns a new entry with the "error" set to `err`.
func (l *Logger) WithError(err error) *Entry {
	return NewEntry(l).WithError(err)
}

// Debug level message.
func (l *Logger) Debug(msg string) {
	NewEntry(l).Debug(msg)
}

// Info level message.
func (l *Logger) Info(msg string) {
	NewEntry(l).Info(msg)
}

// Warn level message.
func (l *Logger) Warn(msg string) {
	NewEntry(l).Warn(msg)
}

// Error level message.
func (l *Logger) Error(msg string) {
	NewEntry(l).Error(msg)
}

// Fatal level message, followed by an exit.
func (l *Logger) Fatal(msg string) {
	NewEntry(l).Fatal(msg)
}

// Panic level message, followed by a panic.
func (l *Logger) Panic(msg string) {
	NewEntry(l).Panic(msg)
}

// IfError returns an Interface that will only log if err is not nil
func (l *Logger) IfError(err error) Interface {
	if err != nil {
		return l.WithError(err)
	}

	return Nil
}

// Trace returns a new entry with a Stop method to fire off
// a corresponding completion log, useful with defer.
func (l *Logger) Trace(level Level, msg string) *Entry {
	return NewEntry(l).Trace(level, msg)
}

// log the message, invoking the handler. We clone the entry here
// to bypass the overhead in Entry methods when the level is not
// met.
func (l *Logger) log(level Level, e *Entry, msg string) {
	handlers := l.handlers[level]
	if len(handlers) == 0 {
		return
	}

	e = e.finalize(level, msg)
	for _, h := range handlers {
		if err := h.HandleLog(e); err != nil {
			log.Printf("error logging: %s", err)
		}
	}
}

// Nil logger that satisfies zlog.Interface but sends all messages to the bit
// bucket
var Nil = &Logger{}

// assert interface compliance.
var _ Interface = Nil
