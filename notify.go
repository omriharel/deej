package deej

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/omriharel/deej/icon"
	"github.com/omriharel/deej/util"
)

// Notifier provides generic notification sending
type Notifier interface {
	Notify(title string, message string) error
}

// ToastNotifier provides toast notifications for Windows
type ToastNotifier struct{}

// NewToastNotifier creates a new ToastNotifier
func NewToastNotifier() (*ToastNotifier, error) {
	return &ToastNotifier{}, nil
}

// Notify sends a toast notification (or falls back to other types of notification for older Windows versions)
func (tn *ToastNotifier) Notify(title string, message string) error {

	// we need to unpack deej.ico somewhere to remain portable. we already have it as bytes so it should be fine
	appIconPath := filepath.Join(os.TempDir(), "deej.ico")

	if !util.FileExists(appIconPath) {
		f, err := os.Create(appIconPath)
		if err != nil {
			return fmt.Errorf("create toast notification icon: %w", err)
		}

		if _, err = f.Write(icon.Data); err != nil {
			return fmt.Errorf("write toast notification icon: %w", err)
		}

		if err = f.Close(); err != nil {
			return fmt.Errorf("close toast notification icon: %w", err)
		}
	}

	// send the actual notification
	if err := beeep.Notify(title, message, appIconPath); err != nil {
		return fmt.Errorf("send toast notification: %w", err)
	}

	return nil
}
