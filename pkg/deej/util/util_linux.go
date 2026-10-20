package util

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const niriRequestTimeout = 500 * time.Millisecond

type niriFocusedWindow struct {
	AppID string `json:"app_id"`
	PID   *int   `json:"pid"`
	Title string `json:"title"`
}

type niriReply struct {
	Ok *struct {
		FocusedWindow *niriFocusedWindow `json:"FocusedWindow"`
	} `json:"Ok"`
	Err *string `json:"Err"`
}

// getCurrentWindowProcessNames queries niri's IPC for the focused window and
// returns every plausible process name for it: the process executable, the
// executable's basename, and the window's app_id. This covers native Wayland
// apps as well as XWayland apps (where niri may report xwayland-satellite as
// the PID).
func getCurrentWindowProcessNames() ([]string, error) {
	window, err := getNiriFocusedWindow()
	if err != nil {
		return nil, err
	}

	if window == nil {
		return nil, nil
	}

	names := []string{}

	if window.PID != nil && *window.PID > 0 {
		pid := *window.PID

		if data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
			names = append(names, strings.TrimSpace(string(data)))
		}

		if executable, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil {
			names = append(names, filepath.Base(executable))
		}
	}

	if window.AppID != "" {
		names = append(names, window.AppID)
	}

	return uniqueNonEmpty(names), nil
}

func getNiriFocusedWindow() (*niriFocusedWindow, error) {
	socketPath := os.Getenv("NIRI_SOCKET")
	if socketPath == "" {
		socketPath = findNiriSocket()
	}
	if socketPath == "" {
		return nil, fmt.Errorf("NIRI_SOCKET is not set and no niri socket was found")
	}

	conn, err := net.DialTimeout("unix", socketPath, niriRequestTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect to niri IPC: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(niriRequestTimeout)); err != nil {
		return nil, fmt.Errorf("set niri IPC deadline: %w", err)
	}

	if _, err := conn.Write([]byte("\"FocusedWindow\"\n")); err != nil {
		return nil, fmt.Errorf("write niri IPC request: %w", err)
	}

	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read niri IPC reply: %w", err)
	}

	var reply niriReply
	if err := json.Unmarshal(line, &reply); err != nil {
		return nil, fmt.Errorf("parse niri IPC reply: %w", err)
	}

	if reply.Err != nil {
		return nil, fmt.Errorf("niri IPC error: %s", *reply.Err)
	}

	if reply.Ok == nil {
		return nil, fmt.Errorf("unexpected niri IPC reply: %s", strings.TrimSpace(string(line)))
	}

	return reply.Ok.FocusedWindow, nil
}

func findNiriSocket() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		return ""
	}

	matches, err := filepath.Glob(filepath.Join(runtimeDir, "niri.wayland-*.sock"))
	if err != nil || len(matches) == 0 {
		return ""
	}

	return matches[0]
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
