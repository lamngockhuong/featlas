package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lamngockhuong/muninn/internal/content"
)

func newValidateCmd() *cobra.Command {
	var dir, format string
	cmd := &cobra.Command{
		Use:   "validate [dir]",
		Short: "Check schema, references, images, hotspots and sources; exit 1 on errors",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				dir = resolveContentDir(args[0])
			}
			ix, err := content.Load(dir)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			errs := ix.ErrorCount()
			switch format {
			case "json":
				issues := ix.Issues
				if issues == nil {
					issues = []content.Issue{}
				}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if err := enc.Encode(map[string]any{"errors": errs, "warnings": len(ix.Issues) - errs, "issues": issues}); err != nil {
					return err
				}
			case "text":
				for _, is := range ix.Issues {
					fmt.Fprintln(out, is.String())
				}
				fmt.Fprintf(out, "%d entities, %d errors, %d warnings\n", len(ix.All()), errs, len(ix.Issues)-errs)
			default:
				return fmt.Errorf("unknown format %q (want text or json)", format)
			}
			if errs > 0 {
				return &exitError{code: 1}
			}
			return nil
		},
	}
	addDirFlag(cmd, &dir)
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}

// resolveContentDir accepts either the content directory itself or a project
// directory that contains muninn/muninn.yaml.
func resolveContentDir(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, content.ConfigFile)); err == nil {
		return dir
	}
	if _, err := os.Stat(filepath.Join(dir, defaultDir, content.ConfigFile)); err == nil {
		return filepath.Join(dir, defaultDir)
	}
	return dir
}
