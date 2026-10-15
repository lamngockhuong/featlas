package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lamngockhuong/muninn/internal/review"
)

// StatusMarker identifies the PR comment written from the markdown format,
// so CI can update its previous comment instead of adding a new one.
const StatusMarker = "<!-- muninn-status -->"

type statusRow struct {
	Key     string          `json:"key"`
	Kind    string          `json:"kind"`
	ID      string          `json:"id"`
	Title   string          `json:"title"`
	File    string          `json:"file"`
	State   string          `json:"state"`
	Reasons []review.Reason `json:"reasons,omitempty"`
	Changed []string        `json:"changed,omitempty"`
}

func newStatusCmd() *cobra.Command {
	var dir, format, since string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "List entities by computed review state",
		Long: "List entities by computed review state: draft, changes_requested, modified (edited after approval),\n" +
			"outdated (sources changed after approval) and approved.\n" +
			"--changed-since keeps only entities whose sources differ from a git ref.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			l, err := load(dir)
			if err != nil {
				return err
			}
			var changed []string
			if since != "" {
				if changed, err = review.ChangedFiles(l.ix.RepoRoot, since); err != nil {
					return err
				}
			}
			lang := l.ix.DefaultLanguage()
			var rows []statusRow
			for _, e := range l.ix.All() {
				row := statusRow{Key: e.Key(), Kind: string(e.Kind), ID: e.ID, Title: e.Title(lang, lang), File: e.File}
				st := l.statuses[e.Key()]
				row.State, row.Reasons = st.State, st.Reasons
				if since != "" {
					row.Changed = review.TouchedSources(e, changed)
					if len(row.Changed) == 0 {
						continue
					}
				}
				rows = append(rows, row)
			}
			out := cmd.OutOrStdout()
			switch format {
			case "text":
				writeStatusText(out, rows, since != "")
			case "json":
				if rows == nil {
					rows = []statusRow{}
				}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{"counts": countStates(rows), "entities": rows})
			case "markdown":
				writeStatusMarkdown(out, rows, since)
			default:
				return fmt.Errorf("unknown format %q (want text, json or markdown)", format)
			}
			return nil
		},
	}
	addDirFlag(cmd, &dir)
	cmd.Flags().StringVar(&format, "format", "text", "output format: text, json or markdown")
	cmd.Flags().StringVar(&since, "changed-since", "", "only entities whose sources changed since this git ref")
	return cmd
}

func countStates(rows []statusRow) map[string]int {
	c := map[string]int{}
	for _, s := range review.States {
		c[s] = 0
	}
	for _, r := range rows {
		c[r.State]++
	}
	return c
}

func reasonText(r review.Reason) string {
	if r.Path == "" {
		return r.Kind
	}
	return r.Kind + " " + r.Path
}

func writeStatusText(w io.Writer, rows []statusRow, withChanged bool) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, state := range review.States {
		for _, r := range rows {
			if r.State != state {
				continue
			}
			detail := make([]string, 0, len(r.Reasons))
			for _, re := range r.Reasons {
				detail = append(detail, reasonText(re))
			}
			if withChanged {
				detail = append(detail, "changed: "+strings.Join(r.Changed, ", "))
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.State, r.Key, r.Title, strings.Join(detail, "; "))
		}
	}
	tw.Flush()
	c := countStates(rows)
	parts := make([]string, 0, len(review.States))
	for _, s := range review.States {
		parts = append(parts, fmt.Sprintf("%d %s", c[s], s))
	}
	fmt.Fprintf(w, "%d entities: %s\n", len(rows), strings.Join(parts, ", "))
}

// writeStatusMarkdown renders the PR comment body.
func writeStatusMarkdown(w io.Writer, rows []statusRow, since string) {
	fmt.Fprintln(w, StatusMarker)
	if since != "" {
		if len(rows) == 0 {
			fmt.Fprintln(w, "### muninn: no business pages affected")
			fmt.Fprintln(w, "\nThis change does not touch any source listed by a feature, screen, flow or module.")
			return
		}
		fmt.Fprintf(w, "### muninn: %d business page(s) to re-check\n\n", len(rows))
		fmt.Fprintln(w, "This change edits code that these pages were written from. Ask their reviewer to re-check them after merge.")
		fmt.Fprintln(w, "\n| Page | Review state | Changed files |\n|---|---|---|")
		for _, r := range rows {
			fmt.Fprintf(w, "| %s `%s` %s | %s | %s |\n", r.Kind, r.ID, mdEscape(r.Title), r.State, codeList(r.Changed))
		}
		return
	}
	c := countStates(rows)
	fmt.Fprintf(w, "### muninn: %d pages, %d approved\n\n| Page | Review state | Reasons |\n|---|---|---|\n", len(rows), c[review.StateApproved])
	for _, r := range rows {
		reasons := make([]string, 0, len(r.Reasons))
		for _, re := range r.Reasons {
			reasons = append(reasons, reasonText(re))
		}
		fmt.Fprintf(w, "| %s `%s` %s | %s | %s |\n", r.Kind, r.ID, mdEscape(r.Title), r.State, mdEscape(strings.Join(reasons, "; ")))
	}
}

func codeList(items []string) string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = "`" + strings.ReplaceAll(s, "`", "") + "`"
	}
	return strings.Join(out, ", ")
}

func mdEscape(s string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ").Replace(s)
}
