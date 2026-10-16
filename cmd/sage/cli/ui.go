package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"sage/internal/graph"
	"sage/internal/ui"

	"github.com/spf13/cobra"
)

var (
	uiPort    int
	noBrowser bool
)

type uiOptions struct {
	Port      int
	Host      string
	NoBrowser bool
	DBPath    string
}

func findAvailablePort(host string, startingPort int) (int, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	for p := startingPort; p < startingPort+100; p++ {
		addr := fmt.Sprintf("%s:%d", host, p)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			_ = ln.Close()
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available port found in range %d-%d on host %s", startingPort, startingPort+100, host)
}

func browserCommand(goos string, targetURL string) (string, []string, error) {
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

var openBrowser = func(targetURL string) error {
	cmdName, args, err := browserCommand(runtime.GOOS, targetURL)
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdName, args...)
	return cmd.Start()
}

func runUIWithContext(ctx context.Context, opts uiOptions, out io.Writer) error {
	path := opts.DBPath
	if path == "" {
		path = dbPath
	}
	if path == "" {
		path = cfg.DBPath
	}
	if path == "" {
		path = ".github/sage"
	}

	engine, err := graph.NewEngine(path)
	if err != nil {
		return fmt.Errorf("unable to open embedded SQLite database at %s: %w", path, err)
	}
	defer engine.Close()

	host := opts.Host
	if host == "" {
		host = "127.0.0.1"
	}

	startingPort := opts.Port
	if startingPort == 0 {
		startingPort = 8080
	}

	port, err := findAvailablePort(host, startingPort)
	if err != nil {
		return fmt.Errorf("failed to allocate port: %w", err)
	}

	srv := ui.NewServer(ui.ServerConfig{
		Port:   port,
		Host:   host,
		Engine: engine,
	})

	url := fmt.Sprintf("http://%s:%d/", host, port)
	fmt.Fprintf(out, "Sage Architecture Visualizer listening on %s\n", url)

	if !opts.NoBrowser {
		fmt.Fprintln(out, "Opening default browser... (use --no-browser to disable)")
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(out, "Notice: could not launch browser automatically: %v\n", err)
		}
	}
	fmt.Fprintln(out, "Press Ctrl+C to stop.")

	return srv.Start(ctx)
}

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the interactive architecture visualizer web UI",
	Long: `Launch the local interactive architecture visualizer web UI.
Embeds the web assets and starts a local HTTP server displaying architectural
domain clusters, file inspection drawer, and decision lineage.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		opts := uiOptions{
			Port:      uiPort,
			Host:      "127.0.0.1",
			NoBrowser: noBrowser,
			DBPath:    dbPath,
		}
		return runUIWithContext(ctx, opts, cmd.OutOrStdout())
	},
}

func init() {
	uiCmd.Flags().IntVarP(&uiPort, "port", "p", 8080, "Port to run the UI server on (default: 8080 or next open port)")
	uiCmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not automatically launch the browser")
}
