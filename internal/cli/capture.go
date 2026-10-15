package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/lamngockhuong/muninn/internal/capture"
)

func init() {
	extraCommands = append(extraCommands, func(string) *cobra.Command { return newCaptureCmd() }, func(string) *cobra.Command { return newValidateCaptureCmd() }, func(string) *cobra.Command { return newFinalizeCmd() })
}

func newCaptureCmd() *cobra.Command {
	var script, chrome, profile string
	var headful bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "capture --script capture.yaml",
		Short: "Take screenshots with element boxes using Chrome (chromedp backend)",
		Long: "Run a capture script with a locally installed Chrome or Chromium. Each capture step writes\n" +
			"<out>.webp and <out>.capture.json, measured and shot in the same DOM state.\n" +
			"Secrets come from ${env:NAME} references; missing variables fail before the browser starts.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := capture.LoadScript(script)
			if err != nil {
				return fmt.Errorf("%s: %w", script, err)
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()
			files, err := capture.Run(ctx, s, filepath.Dir(script), capture.Options{
				Chrome: chrome, Profile: profile, Headful: headful, Timeout: timeout, Log: cmd.ErrOrStderr(),
			})
			for _, f := range files {
				fmt.Fprintln(cmd.OutOrStdout(), f)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&script, "script", "capture.yaml", "capture script")
	cmd.Flags().StringVar(&chrome, "chrome", "", "Chrome or Chromium executable (default: auto-detect)")
	cmd.Flags().StringVar(&profile, "profile", "", "browser profile directory kept between runs, e.g. .muninn-profile (keep it out of git)")
	cmd.Flags().BoolVar(&headful, "headful", false, "show the browser window")
	cmd.Flags().DurationVar(&timeout, "step-timeout", 30*time.Second, "timeout per step")
	return cmd
}

func newValidateCaptureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate-capture <file.capture.json>...",
		Short: "Check capture.json files against the schema and their images",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bad := 0
			for _, f := range args {
				problems, err := capture.ValidateFile(f)
				if err != nil {
					return fmt.Errorf("%s: %w", f, err)
				}
				for _, p := range problems {
					fmt.Fprintf(cmd.OutOrStdout(), "error: %s: %s\n", f, p)
				}
				if len(problems) > 0 {
					bad++
				}
			}
			if bad > 0 {
				return &exitError{code: 1, msg: fmt.Sprintf("%d of %d capture files invalid", bad, len(args))}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d capture files valid\n", len(args))
			return nil
		},
	}
}

func newFinalizeCmd() *cobra.Command {
	var input string
	cmd := &cobra.Command{
		Use:   "capture-finalize --input raw.json",
		Short: "Turn a raw measurement and PNG from another backend into <out>.webp and <out>.capture.json",
		Long:  "Used by the Playwright and pinchtab capture scripts so all backends share one crop, resize and encoding step.\nInput: JSON with backend, url, viewport, dpr, maxWidth, out, screenshot (PNG path), elements (CSS px), crop.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			raw, err := os.ReadFile(input)
			if err != nil {
				return err
			}
			var in capture.Raw
			if err := json.Unmarshal(raw, &in); err != nil {
				return fmt.Errorf("%s: %w", input, err)
			}
			png, err := os.ReadFile(in.Screenshot)
			if err != nil {
				return err
			}
			if _, err := capture.Finalize(in, png); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), in.Out+".capture.json")
			return nil
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "raw measurement JSON")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}
