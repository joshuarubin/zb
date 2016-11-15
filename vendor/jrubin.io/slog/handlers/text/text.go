// Package text implements textual handler suitable for development and
// production output. It will automatically colorize the output if it detects
// that it is attached to a terminal. While possible, it is not suggested to use
// this handler to write to files.
package text

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"jrubin.io/slog"
)

// Default handler outputting to stderr.
var Default = New(os.Stderr)

// Logger returns a logger configured to output text at level or higher to
// stderr.
func Logger(level slog.Level) *slog.Logger {
	return slog.New().RegisterHandler(level, Default)
}

var (
	start      = time.Now()
	isTerminal = IsTerminal()
)

// colors.
const (
	red    = 31
	yellow = 33
	blue   = 34
	gray   = 37
)

// Colors mapping.
var Colors = [...]int{
	slog.DebugLevel: gray,
	slog.InfoLevel:  blue,
	slog.WarnLevel:  yellow,
	slog.ErrorLevel: red,
	slog.FatalLevel: red,
	slog.PanicLevel: red,
}

// field used for sorting.
type field struct {
	Name  string
	Value interface{}
}

// by sorts projects by call count.
type byName []field

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }

// Handler implementation.
type Handler struct {
	mu               sync.Mutex
	Writer           io.Writer
	ForceColors      bool
	DisableColors    bool
	DisableTimestamp bool
	FullTimestamp    bool
	DisableSorting   bool
	TimestampFormat  string
}

// New handler.
func New(w io.Writer) *Handler {
	return &Handler{
		Writer: w,
	}
}

// DefaultTimestampFormat is used when FullTimestamp is empty and the
// application is not connected to a terminal or FullTimestamp is true.
const DefaultTimestampFormat = time.RFC3339

// HandleLog implements slog.Handler.
func (h *Handler) HandleLog(e *slog.Entry) error {
	var fields []field

	for k, v := range e.Fields {
		fields = append(fields, field{k, v})
	}

	if !h.DisableSorting {
		sort.Sort(byName(fields))
	}

	isColorTerminal := isTerminal && (runtime.GOOS != "windows")
	isColored := (h.ForceColors || isColorTerminal) && !h.DisableColors

	timestampFormat := h.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = DefaultTimestampFormat
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if isColored {
		h.printColored(e, fields, timestampFormat)
	} else {
		if !h.DisableTimestamp {
			h.appendKeyValue("time", e.Time.Format(timestampFormat))
		}

		h.appendKeyValue("level", strings.ToUpper(e.Level.String()))

		if e.Message != "" {
			h.appendKeyValue("msg", e.Message)
		}

		for _, f := range fields {
			h.appendKeyValue(f.Name, f.Value)
		}
	}

	fmt.Fprintln(h.Writer)
	return nil
}

func (h *Handler) printColored(e *slog.Entry, fields []field, timestampFormat string) {
	color := Colors[e.Level]

	if h.DisableTimestamp {
		fmt.Fprintf(h.Writer, "\033[%dm%5s\033[0m %-25s", color, strings.ToUpper(e.Level.String()), e.Message)
	} else if !h.FullTimestamp {
		ts := time.Since(start) / time.Second
		fmt.Fprintf(h.Writer, "\033[%dm%5s\033[0m[%04d] %-25s", color, strings.ToUpper(e.Level.String()), ts, e.Message)
	} else {
		fmt.Fprintf(h.Writer, "\033[%dm%5s\033[0m[%s] %-25s", color, strings.ToUpper(e.Level.String()), e.Time.Format(timestampFormat), e.Message)
	}

	for _, f := range fields {
		fmt.Fprintf(h.Writer, " \033[%dm%s\033[0m=%+v", color, f.Name, f.Value)
	}
}

func (h *Handler) appendKeyValue(key string, value interface{}) {
	fmt.Fprintf(h.Writer, "%s=", key)

	switch value := value.(type) {
	case string:
		if !needsQuoting(value) {
			fmt.Fprint(h.Writer, value)
		} else {
			fmt.Fprintf(h.Writer, "%q", value)
		}
	case error:
		errmsg := value.Error()
		if !needsQuoting(errmsg) {
			fmt.Fprint(h.Writer, errmsg)
		} else {
			fmt.Fprintf(h.Writer, "%q", value)
		}
	default:
		fmt.Fprint(h.Writer, value)
	}

	fmt.Fprint(h.Writer, " ")
}

func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return true
		}
	}
	return false
}
