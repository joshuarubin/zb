package slog

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

// Level of severity.
type Level int

// Log levels.
const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

var levelNames = [...]string{
	PanicLevel: "panic",
	FatalLevel: "fatal",
	ErrorLevel: "error",
	WarnLevel:  "warn",
	InfoLevel:  "info",
	DebugLevel: "debug",
}

// Set implements cli.Generic
func (l *Level) Set(value string) error {
	*l = ParseLevel(value, WarnLevel)
	return nil
}

// String implements io.Stringer.
func (l Level) String() string {
	if l < PanicLevel {
		l = PanicLevel
	}

	if l > DebugLevel {
		l = DebugLevel
	}

	return levelNames[l]
}

// MarshalJSON returns the level string.
func (l Level) MarshalJSON() ([]byte, error) {
	return []byte(`"` + l.String() + `"`), nil
}

// MarshalText implements encoding.TextMarshaler
func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (l *Level) UnmarshalText(text []byte) error {
	*l = ParseLevel(string(text), WarnLevel)
	return nil
}

// ParseLevel parses level string.
func ParseLevel(s string, defaultLevel Level) Level {
	if len(s) == 0 {
		return defaultLevel
	}

	if i, err := strconv.Atoi(s); err == nil {
		l := Level(i)

		if l < PanicLevel {
			l = PanicLevel
		}

		if l > DebugLevel {
			l = DebugLevel
		}

		return l
	}

	r, _ := utf8.DecodeRuneInString(s)
	r = unicode.ToLower(r)

	switch r {
	case 'd':
		return DebugLevel
	case 'i':
		return InfoLevel
	case 'w':
		return WarnLevel
	case 'e':
		return ErrorLevel
	case 'f':
		return FatalLevel
	case 'p':
		return PanicLevel
	}

	return defaultLevel
}
