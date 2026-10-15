package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lamngockhuong/muninn/internal/content"
)

const configTemplate = `muninn: 1
project:
  name: My Product
  summary: { en: "One sentence about what the product does and for whom." }
languages: { default: en, available: [en] }
repo:
  root: ..            # repository root, relative to this file; source paths resolve against it
  # url: https://github.com/<owner>/<repo>
  # branch: main
roles:
  - { id: user, label: { en: User } }
areas:                # group features; name them after the app's menu
  - { id: general, label: { en: General } }
# owners:             # default reviewers per area
#   general: [your-name]
`

func newInitCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a muninn.yaml template and empty content directories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := filepath.Join(dir, content.ConfigFile)
			if _, err := os.Stat(cfg); err == nil {
				return fmt.Errorf("%s already exists", cfg)
			}
			for _, k := range content.Kinds {
				if err := os.MkdirAll(filepath.Join(dir, k.Dir()), 0o755); err != nil {
					return err
				}
			}
			if err := os.WriteFile(cfg, []byte(configTemplate), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s\nnext: edit roles and areas, generate content, then run `muninn validate --dir %s`\n", cfg, dir)
			return nil
		},
	}
	addDirFlag(cmd, &dir)
	return cmd
}
