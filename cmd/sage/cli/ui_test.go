package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"sage/cmd/sage/cli"
)

func TestBrowserCommand(t *testing.T) {
	tests := []struct {
		goos      string
		url       string
		wantCmd   string
		wantArgs  []string
		wantErr   bool
	}{
		{
			goos:     "linux",
			url:      "http://127.0.0.1:8080",
			wantCmd:  "xdg-open",
			wantArgs: []string{"http://127.0.0.1:8080"},
			wantErr:  false,
		},
		{
			goos:     "darwin",
			url:      "http://127.0.0.1:8080",
			wantCmd:  "open",
			wantArgs: []string{"http://127.0.0.1:8080"},
			wantErr:  false,
		},
		{
			goos:     "windows",
			url:      "http://127.0.0.1:8080",
			wantCmd:  "rundll32",
			wantArgs: []string{"url.dll,FileProtocolHandler", "http://127.0.0.1:8080"},
			wantErr:  false,
		},
		{
			goos:    "unknown-os",
			url:     "http://127.0.0.1:8080",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.goos, func(t *testing.T) {
			cmdName, args, err := cli.BrowserCommand(tc.goos, tc.url)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for goos %s, got nil", tc.goos)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.goos, err)
			}
			if cmdName != tc.wantCmd {
				t.Errorf("cmdName = %q, want %q", cmdName, tc.wantCmd)
			}
			if len(args) != len(tc.wantArgs) {
				t.Fatalf("args len = %d, want %d", len(args), len(tc.wantArgs))
			}
			for i := range args {
				if args[i] != tc.wantArgs[i] {
					t.Errorf("arg[%d] = %q, want %q", i, args[i], tc.wantArgs[i])
				}
			}
		})
	}
}

func TestFindAvailablePort_SelectsAvailablePort(t *testing.T) {
	// Find a free port starting at 19000
	port, err := cli.FindAvailablePort("127.0.0.1", 19000)
	if err != nil {
		t.Fatalf("unexpected error finding port: %v", err)
	}
	if port < 19000 {
		t.Errorf("expected port >= 19000, got %d", port)
	}
}

func TestFindAvailablePort_BypassesOccupiedPort(t *testing.T) {
	// Occupy port 19100
	basePort := 19100
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", basePort))
	if err != nil {
		t.Skipf("cannot bind test port %d: %v", basePort, err)
	}
	defer ln.Close()

	port, err := cli.FindAvailablePort("127.0.0.1", basePort)
	if err != nil {
		t.Fatalf("unexpected error finding port: %v", err)
	}
	if port <= basePort {
		t.Errorf("expected port > %d (occupied), got %d", basePort, port)
	}
}

func TestUICmd_DefaultFlagsAndExecution(t *testing.T) {
	var openedURL string
	var serverStarted bool
	var capturedHost string
	var capturedPort int
	var capturedFile string

	// Mock browser opener
	cli.SetBrowserOpenerForTest(func(url string) error {
		openedURL = url
		return nil
	})
	defer cli.ResetBrowserOpenerForTest()

	// Mock server runner
	cli.SetServerRunnerForTest(func(ctx context.Context, host string, port int, filePath string) error {
		serverStarted = true
		capturedHost = host
		capturedPort = port
		capturedFile = filePath
		return nil
	})
	defer cli.ResetServerRunnerForTest()

	cmd := cli.NewUICmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--no-browser"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected successful execution, got: %v", err)
	}

	if !serverStarted {
		t.Error("expected server to be started")
	}
	if capturedHost != "127.0.0.1" {
		t.Errorf("expected default host 127.0.0.1, got %s", capturedHost)
	}
	if capturedPort != 8080 {
		t.Errorf("expected default port 8080, got %d", capturedPort)
	}
	if capturedFile != "SAGE.md" {
		t.Errorf("expected default file SAGE.md, got %s", capturedFile)
	}
	if openedURL != "" {
		t.Errorf("expected no browser launch with --no-browser, but opened: %s", openedURL)
	}

	output := out.String()
	if !strings.Contains(output, "Sage UI server started at http://127.0.0.1:8080") {
		t.Errorf("expected output to contain start URL, got:\n%s", output)
	}
}

func TestUICmd_OpensBrowserWhenNotDisabled(t *testing.T) {
	var openedURL string
	cli.SetBrowserOpenerForTest(func(url string) error {
		openedURL = url
		return nil
	})
	defer cli.ResetBrowserOpenerForTest()

	cli.SetServerRunnerForTest(func(ctx context.Context, host string, port int, filePath string) error {
		return nil
	})
	defer cli.ResetServerRunnerForTest()

	cmd := cli.NewUICmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--port", "8085", "--file", "custom.md"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected successful execution, got: %v", err)
	}

	expectedURL := "http://127.0.0.1:8085"
	if openedURL != expectedURL {
		t.Errorf("expected browser to open %q, got %q", expectedURL, openedURL)
	}
}

func TestUICmd_HandlesPortConflictAndResolvesNext(t *testing.T) {
	basePort := 19200
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", basePort))
	if err != nil {
		t.Skipf("cannot bind test port %d: %v", basePort, err)
	}
	defer ln.Close()

	var actualPort int
	cli.SetServerRunnerForTest(func(ctx context.Context, host string, port int, filePath string) error {
		actualPort = port
		return nil
	})
	defer cli.ResetServerRunnerForTest()

	cmd := cli.NewUICmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--port", fmt.Sprintf("%d", basePort), "--no-browser"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected successful execution, got: %v", err)
	}

	if actualPort <= basePort {
		t.Errorf("expected resolved port > %d, got %d", basePort, actualPort)
	}

	output := out.String()
	expectedNotice := fmt.Sprintf("Sage UI server started at http://127.0.0.1:%d", actualPort)
	if !strings.Contains(output, expectedNotice) {
		t.Errorf("expected output to mention %s, got:\n%s", expectedNotice, output)
	}
}
