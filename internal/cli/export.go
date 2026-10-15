package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lamngockhuong/muninn/internal/export"
	"github.com/lamngockhuong/muninn/internal/server"
	"github.com/lamngockhuong/muninn/internal/webui"
)

func init() {
	extraCommands = append(extraCommands, newExportCmd)
}

func newExportCmd(version string) *cobra.Command {
	var dir, out string
	cmd := &cobra.Command{
		Use:   "export --out site/",
		Short: "Write a read-only static site (web app, JSON data, images) for GitHub Pages or any static host",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snap, err := server.NewSnapshot(dir, nil, 1)
			if err != nil {
				return err
			}
			res, err := export.Write(snap, export.Options{Out: out, UI: webui.Files(), Version: version})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exported %d entities, %d images, %d files to %s\n", res.Entities, res.Images, res.Files, out)
			if n := snap.Index.ErrorCount(); n > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: content has %d validation errors (see `muninn validate`)\n", n)
			}
			return nil
		},
	}
	addDirFlag(cmd, &dir)
	cmd.Flags().StringVar(&out, "out", "site", "output directory (must be empty or a previous export)")
	return cmd
}
