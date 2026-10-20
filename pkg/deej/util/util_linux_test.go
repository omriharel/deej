package util

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
)

func TestGetCurrentWindowProcessNamesFromNiri(t *testing.T) {
	dir, err := ioutil.TempDir("", "deej-niri-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	socketPath := dir + "/niri.sock"

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on fake niri socket: %v", err)
	}
	defer listener.Close()

	requestReceived := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return
		}
		requestReceived <- strings.TrimSpace(line)

		response := fmt.Sprintf(`{"Ok":{"FocusedWindow":{"app_id":"test-app","pid":%d,"title":"Watch_Dogs 2"}}}`, os.Getpid())
		conn.Write([]byte(response + "\n"))
	}()

	oldSocket, hadSocket := os.LookupEnv("NIRI_SOCKET")
	if err := os.Setenv("NIRI_SOCKET", socketPath); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if hadSocket {
			os.Setenv("NIRI_SOCKET", oldSocket)
		} else {
			os.Unsetenv("NIRI_SOCKET")
		}
	}()

	names, err := GetCurrentWindowProcessNames()
	if err != nil {
		t.Fatalf("GetCurrentWindowProcessNames: %v", err)
	}

	select {
	case request := <-requestReceived:
		if request != `"FocusedWindow"` {
			t.Errorf("unexpected niri request: %q", request)
		}
	default:
		t.Error("niri request was not received")
	}

	foundAppID := false
	foundNormalizedTitle := false
	for _, name := range names {
		if name == "test-app" {
			foundAppID = true
		}
		if name == "watchdogs2" {
			foundNormalizedTitle = true
		}
	}
	if !foundAppID {
		t.Errorf("expected app_id in process names, got %v", names)
	}
	if !foundNormalizedTitle {
		t.Errorf("expected normalized title in process names, got %v", names)
	}
}
