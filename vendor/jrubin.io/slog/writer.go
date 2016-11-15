package slog

import (
	"bytes"
	"runtime"
	"sync"
)

type syncWriter struct {
	prefix    string
	printFunc func(string)
	buf       bytes.Buffer
	mu        sync.Mutex
}

func (w *syncWriter) Prefix(prefix string) PrefixWriteCloser {
	w.mu.Lock()
	w.prefix += prefix
	w.mu.Unlock()
	return w
}

func (w *syncWriter) printLines() {
	for {
		i := bytes.IndexByte(w.buf.Bytes(), '\n')
		if i < 0 {
			break
		}

		data := w.buf.Next(i + 1)

		// strip trailing "\r\n"
		for _, b := range []byte("\n\r") { // yes, I know "\n\r" is backwards
			if len(data) > 0 && data[len(data)-1] == b {
				data = data[:len(data)-1]
			}
		}

		w.print(string(data))
	}
}

func (w *syncWriter) print(data string) {
	if len(data) > 0 {
		w.printFunc(w.prefix + data)
	}
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	n, err := w.buf.Write(p)
	w.printLines()
	w.mu.Unlock()

	return n, err
}

func (w *syncWriter) Close() error {
	w.mu.Lock()
	w.printLines()
	w.print(w.buf.String())
	w.buf.Reset()
	w.mu.Unlock()
	return nil
}

// Writer returns an io.WriteCloser where each line written to that writer will
// be printed using the handlers for the given Level. It is the caller's
// responsibility to close it.
func (l *Logger) Writer(level Level) PrefixWriteCloser {
	return NewEntry(l).Writer(level)
}

// Writer returns an io.WriteCloser where each line written to that writer will
// be printed using the handlers for the given Level. It is the caller's
// responsibility to close it.
func (e *Entry) Writer(level Level) PrefixWriteCloser {
	if level < PanicLevel {
		level = PanicLevel
	}

	if level > DebugLevel {
		level = DebugLevel
	}

	var printFunc func(msg string)
	switch level {
	case DebugLevel:
		printFunc = e.Debug
	case InfoLevel:
		printFunc = e.Info
	case WarnLevel:
		printFunc = e.Warn
	case ErrorLevel:
		printFunc = e.Error
	case FatalLevel:
		printFunc = e.Fatal
	case PanicLevel:
		printFunc = e.Panic
	}

	w := &syncWriter{
		printFunc: printFunc,
	}

	runtime.SetFinalizer(w, writerFinalizer)

	return w
}

func writerFinalizer(writer *syncWriter) {
	_ = writer.Close()
}
