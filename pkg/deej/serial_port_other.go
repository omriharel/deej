//go:build !linux
// +build !linux

package deej

import (
	"fmt"
	"strings"
)

func resolveSerialPort(configured, wantVID, wantPID string) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured == "" || strings.EqualFold(configured, autoCOMPort) {
		return "", fmt.Errorf("automatic serial port detection is only supported on Linux; set com_port explicitly")
	}

	return configured, nil
}
