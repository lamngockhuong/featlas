package cli

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/lamngockhuong/muninn/internal/server"
	"github.com/lamngockhuong/muninn/internal/webui"
)

func init() {
	extraCommands = append(extraCommands, newServeCmd)
}

func newServeCmd(version string) *cobra.Command {
	var dir, host, token string
	var port int
	var readOnly, open, verbose bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the web UI and API with live reload",
		Long: "Serve the web UI and API. Content and source changes rebuild the index and refresh open pages.\n" +
			"Binding outside localhost without --token switches to read-only mode.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			level := slog.LevelInfo
			if verbose {
				level = slog.LevelDebug
			}
			logger := slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{Level: level}))
			if token == "" {
				token = os.Getenv("MUNINN_TOKEN")
			}
			local := server.IsLoopback(host)
			reason := ""
			if readOnly {
				reason = "started with --read-only"
			} else if !local && token == "" {
				readOnly = true
				reason = "bound outside localhost without a token"
				logger.Warn("binding outside localhost without --token: reviewing is disabled (read-only)", "host", host)
			}
			srv, err := server.New(server.Options{
				Dir: dir, ReadOnly: readOnly, ReadOnlyReason: reason, Token: token, Local: local, Host: host,
				Version: version, UI: webui.Handler(), Logger: logger,
			})
			if err != nil {
				return err
			}
			ln, err := server.Listen(host, port)
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			go func() {
				if err := srv.Watch(ctx); err != nil {
					logger.Error("file watcher stopped", "error", err)
				}
			}()
			url := "http://" + net.JoinHostPort(displayHost(host), strconv.Itoa(port))
			snap := srv.Snapshot()
			logger.Info("muninn serving", "url", url, "dir", snap.Index.Dir, "entities", len(snap.Statuses), "errors", snap.Index.ErrorCount(), "readOnly", readOnly)
			if open {
				openBrowser(url, logger)
			}
			err = srv.Serve(ctx, ln)
			logger.Info("stopped")
			return err
		},
	}
	addDirFlag(cmd, &dir)
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "address to bind")
	cmd.Flags().IntVar(&port, "port", 7077, "port to bind")
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "disable reviewing")
	cmd.Flags().StringVar(&token, "token", "", "require this bearer token for reviews (or set MUNINN_TOKEN)")
	cmd.Flags().BoolVar(&open, "open", false, "open the browser")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "log every request")
	return cmd
}

func displayHost(host string) string {
	if host == "" || host == "0.0.0.0" || host == "::" {
		return "localhost"
	}
	return host
}

func openBrowser(url string, logger *slog.Logger) {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	if err := c.Start(); err != nil {
		logger.Warn("cannot open the browser", "error", err)
		return
	}
	go func() { _ = c.Wait() }()
}
