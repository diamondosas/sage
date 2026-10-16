package cli

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestUICommandRegistration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"ui"})
	if err != nil || cmd == nil || cmd.Name() != "ui" {
		t.Fatalf("expected 'ui' subcommand to be registered on rootCmd")
	}

	portFlag := cmd.Flag("port")
	if portFlag == nil {
		t.Fatalf("expected --port flag to be registered on ui command")
	}
	if portFlag.DefValue != "8080" {
		t.Errorf("expected default port to be 8080, got %s", portFlag.DefValue)
	}

	noBrowserFlag := cmd.Flag("no-browser")
	if noBrowserFlag == nil {
		t.Fatalf("expected --no-browser flag to be registered on ui command")
	}
	if noBrowserFlag.DefValue != "false" {
		t.Errorf("expected default no-browser to be false, got %s", noBrowserFlag.DefValue)
	}
}

func TestFindAvailablePort(t *testing.T) {
	// First, test when starting port is available
	port, err := findAvailablePort("127.0.0.1", 19280)
	if err != nil {
		t.Fatalf("expected port to be available, got error: %v", err)
	}
	if port != 19280 {
		t.Errorf("expected port 19280, got %d", port)
	}

	// Now occupy port 19280 and verify it returns next available port (19281)
	ln, err := net.Listen("tcp", "127.0.0.1:19280")
	if err != nil {
		t.Fatalf("failed to listen on test port: %v", err)
	}
	defer ln.Close()

	nextPort, err := findAvailablePort("127.0.0.1", 19280)
	if err != nil {
		t.Fatalf("expected next port to be found, got error: %v", err)
	}
	if nextPort <= 19280 {
		t.Errorf("expected port > 19280 when 19280 is occupied, got %d", nextPort)
	}
}

func TestBrowserCommand(t *testing.T) {
	testCases := []struct {
		goos        string
		url         string
		expectedCmd string
		expectedArg string
		expectErr   bool
	}{
		{
			goos:        "linux",
			url:         "http://127.0.0.1:8080",
			expectedCmd: "xdg-open",
			expectedArg: "http://127.0.0.1:8080",
			expectErr:   false,
		},
		{
			goos:        "darwin",
			url:         "http://127.0.0.1:8080",
			expectedCmd: "open",
			expectedArg: "http://127.0.0.1:8080",
			expectErr:   false,
		},
		{
			goos:        "windows",
			url:         "http://127.0.0.1:8080",
			expectedCmd: "rundll32",
			expectedArg: "http://127.0.0.1:8080",
			expectErr:   false,
		},
		{
			goos:      "plan9",
			url:       "http://127.0.0.1:8080",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		cmdName, args, err := browserCommand(tc.goos, tc.url)
		if tc.expectErr {
			if err == nil {
				t.Errorf("expected error for OS %q, got nil", tc.goos)
			}
			continue
		}

		if err != nil {
			t.Errorf("unexpected error for OS %q: %v", tc.goos, err)
			continue
		}

		if cmdName != tc.expectedCmd {
			t.Errorf("OS %q: expected cmd %q, got %q", tc.goos, tc.expectedCmd, cmdName)
		}

		found := false
		for _, arg := range args {
			if arg == tc.expectedArg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("OS %q: expected args %v to contain %q", tc.goos, args, tc.expectedArg)
		}
	}
}

func TestRunUIWithContext(t *testing.T) {
	tmpDir := t.TempDir()

	var openedURL string
	origOpenBrowser := openBrowser
	openBrowser = func(targetURL string) error {
		openedURL = targetURL
		return nil
	}
	defer func() { openBrowser = origOpenBrowser }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf := new(bytes.Buffer)
	opts := uiOptions{
		Port:      19290,
		Host:      "127.0.0.1",
		NoBrowser: false,
		DBPath:    tmpDir,
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- runUIWithContext(ctx, opts, buf)
	}()

	// Wait for server to respond
	var serverReady bool
	url := fmt.Sprintf("http://127.0.0.1:%d/", 19290)
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				serverReady = true
				break
			}
		}
	}

	if !serverReady {
		t.Fatalf("server failed to start and respond at %s", url)
	}

	if openedURL != url {
		t.Errorf("expected opened browser URL to be %q, got %q", url, openedURL)
	}

	outStr := buf.String()
	if !strings.Contains(outStr, "Sage Architecture Visualizer listening on http://127.0.0.1:19290") {
		t.Errorf("expected output to mention listening address, got: %s", outStr)
	}

	// Trigger graceful shutdown
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("runUIWithContext returned unexpected error on shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("server did not shut down within timeout")
	}
}

func TestRunUIWithContextNoBrowser(t *testing.T) {
	tmpDir := t.TempDir()

	var browserCalled bool
	origOpenBrowser := openBrowser
	openBrowser = func(targetURL string) error {
		browserCalled = true
		return nil
	}
	defer func() { openBrowser = origOpenBrowser }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buf := new(bytes.Buffer)
	opts := uiOptions{
		Port:      19295,
		Host:      "127.0.0.1",
		NoBrowser: true,
		DBPath:    tmpDir,
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- runUIWithContext(ctx, opts, buf)
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d/", 19295)
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			break
		}
	}

	if browserCalled {
		t.Errorf("browser should NOT be called when NoBrowser is true")
	}

	cancel()
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("runUIWithContext error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("server did not shut down within timeout")
	}
}



