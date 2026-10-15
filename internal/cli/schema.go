package cli

import (
	"github.com/spf13/cobra"

	"github.com/lamngockhuong/muninn/internal/schema"
)

func newSchemaCmd() *cobra.Command {
	var entity string
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Print the JSON Schema of the content format",
		Long:  "Print the JSON Schema of the content format. Generators read it before writing content.\nEntities: config, feature, screen, flow, module, glossary, capture.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			doc, err := schema.Document(entity)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(append(doc, '\n'))
			return err
		},
	}
	cmd.Flags().StringVar(&entity, "entity", "", "print one entity's schema (config, feature, screen, flow, module, glossary, capture)")
	return cmd
}
