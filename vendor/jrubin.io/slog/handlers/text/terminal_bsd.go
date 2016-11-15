// +build darwin freebsd openbsd netbsd dragonfly

package text

import "syscall"

const ioctlReadTermios = syscall.TIOCGETA

// Termios is syscall.Termios
type Termios syscall.Termios
