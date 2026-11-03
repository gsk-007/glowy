//go:build windows

package main

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/sys/windows"
)

func initConsole() {
	handle := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err == nil {
		mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
		_ = windows.SetConsoleMode(handle, mode)
	}

	lipgloss.SetColorProfile(termenv.TrueColor)
	termenv.SetDefaultOutput(termenv.NewOutput(os.Stdout, termenv.WithProfile(termenv.TrueColor)))
}
