package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	internalui "sage/internal/ui"

	"github.com/spf13/cobra"
)

// Default server and launcher configuration values.
// ACTIONABLE: Adjust these constants to change the default listening port,
// default bind host, or port retry attempts across the CLI.
const (
	DefaultPort    = 8080
	DefaultHost    = "127.0.0.1"
	MaxPortRetries = 50
)

// UIOptions holds command-line configuration for the web UI server.
// ACTIONABLE: Add or adjust fields here when introducing new server flags.
type UIOptions struct {
	Port      int
	Host      string
	FilePath  string
	NoBrowser bool
}

// ServerRunner executes an HTTP server until the provided context is canceled.
type ServerRunner func(ctx context.Context, host string, port int, filePath string) error

// BrowserOpener launches the given URL in the user's default browser.
type BrowserOpener func(targetURL string) error

var (
	activeServerRunner  ServerRunner  = defaultServerRunner
	activeBrowserOpener BrowserOpener = openBrowser
)

// SetServerRunnerForTest overrides the active server runner for unit testing.
func SetServerRunnerForTest(runner ServerRunner) {
	activeServerRunner = runner
}

// ResetServerRunnerForTest restores the default server runner.
func ResetServerRunnerForTest() {
	activeServerRunner = defaultServerRunner
}

// SetBrowserOpenerForTest overrides the active browser launcher for unit testing.
func SetBrowserOpenerForTest(opener BrowserOpener) {
	activeBrowserOpener = opener
}

// ResetBrowserOpenerForTest restores the default browser launcher.
func ResetBrowserOpenerForTest() {
	activeBrowserOpener = openBrowser
}

// defaultServerRunner runs the embedded UI server with graceful shutdown.
func defaultServerRunner(ctx context.Context, host string, port int, filePath string) error {
	srv := internalui.NewServer(internalui.Config{
		FilePath: filePath,
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: srv.Handler(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	err := httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// BrowserCommand returns the OS-specific command and arguments for launching a browser.
// ACTIONABLE: To add support for additional platforms or change browser invocation flags,
// edit the cases below.
func BrowserCommand(goos string, targetURL string) (string, []string, error) {
	switch goos {
	case "linux":
		return "xdg-open", []string{targetURL}, nil
	case "darwin":
		return "open", []string{targetURL}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", targetURL}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform %q for opening browser", goos)
	}
}

// openBrowser launches targetURL in the user's desktop browser asynchronously with error logging.
// ACTIONABLE: Adjust browser launching logic, fallback behaviors, or logging outputs here.
func openBrowser(targetURL string) error {
	cmdName, args, err := BrowserCommand(runtime.GOOS, targetURL)
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdName, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		if waitErr := cmd.Wait(); waitErr != nil {
			fmt.Fprintf(os.Stderr, "Notice: browser command exited with error: %v\n", waitErr)
		}
	}()
	return nil
}

// isAddrInUse checks whether an error from net.Listen is due to address/port already in use.
func isAddrInUse(err error) bool {
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.EADDRINUSE) {
			return true
		}
		errLower := strings.ToLower(opErr.Err.Error())
		if strings.Contains(errLower, "address already in use") ||
			strings.Contains(errLower, "only one usage of each socket address") {
			return true
		}
	}
	errLower := strings.ToLower(err.Error())
	return strings.Contains(errLower, "address already in use") ||
		strings.Contains(errLower, "only one usage of each socket address")
}

// FindAvailablePort checks if host:startingPort is open. If occupied by an existing process,
// it probes subsequent ports (port + 1) up to MaxPortRetries times.
func FindAvailablePort(host string, startingPort int) (int, error) {
	if host == "" {
		host = DefaultHost
	}

	for p := startingPort; p < startingPort+MaxPortRetries; p++ {
		addr := fmt.Sprintf("%s:%d", host, p)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			_ = ln.Close()
			return p, nil
		}
		// If the error is NOT address in use (e.g. permission denied or invalid host), fail immediately
		if !isAddrInUse(err) {
			return 0, fmt.Errorf("failed to listen on %s: %w", addr, err)
		}
	}
	return 0, fmt.Errorf("no available port found in range %d-%d on host %s", startingPort, startingPort+MaxPortRetries, host)
}

// RunUIWorkflow orchestrates port fallback, browser launch, and server execution.
func RunUIWorkflow(ctx context.Context, opts UIOptions, out io.Writer) error {
	if opts.Port <= 0 {
		opts.Port = DefaultPort
	}
	if opts.Host == "" {
		opts.Host = DefaultHost
	}
	if opts.FilePath == "" {
		opts.FilePath = DefaultDiagramFile
	}

	actualPort, err := FindAvailablePort(opts.Host, opts.Port)
	if err != nil {
		return fmt.Errorf("port allocation failed: %w", err)
	}

	serverURL := fmt.Sprintf("http://%s:%d", opts.Host, actualPort)
	fmt.Fprintf(out, "Sage UI server started at %s\n", serverURL)

	if !opts.NoBrowser {
		if err := activeBrowserOpener(serverURL); err != nil {
			fmt.Fprintf(out, "Notice: could not launch browser automatically: %v\n", err)
		}
	}

	return activeServerRunner(ctx, opts.Host, actualPort, opts.FilePath)
}

// NewUICmd returns a new Cobra command for "sage ui".
func NewUICmd() *cobra.Command {
	var opts UIOptions

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the interactive architecture visualizer web UI",
		Long:  "Starts the embedded HTTP server, opens the browser, and renders SAGE.md diagrams.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return RunUIWorkflow(ctx, opts, cmd.OutOrStdout())
		},
	}

	cmd.Flags().IntVarP(&opts.Port, "port", "p", DefaultPort, "Port to run the UI server on")
	cmd.Flags().StringVarP(&opts.Host, "host", "H", DefaultHost, "Bind host address")
	cmd.Flags().StringVarP(&opts.FilePath, "file", "f", DefaultDiagramFile, "Path to diagram markdown file")
	cmd.Flags().BoolVar(&opts.NoBrowser, "no-browser", false, "Do not automatically launch the web browser")

	return cmd
}
