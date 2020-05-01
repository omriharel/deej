package util

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// FileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// SetupCloseHandler creates a 'listener' on a new goroutine which will notify the
// program if it receives an interrupt from the OS
func SetupCloseHandler() chan os.Signal {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	return c
}

// OpenExternal spawns a detached window with the provided command and argument
func OpenExternal(logger *zap.SugaredLogger, cmd string, arg string) error {
	command := exec.Command("cmd.exe", "/C", "start", "/b", cmd, arg)

	if err := command.Run(); err != nil {
		logger.Warnw("Failed to spawn detached process",
			"command", cmd,
			"argument", arg,
			"error", err)

		return fmt.Errorf("spawn detached proc: %w", err)
	}

	return nil
}
