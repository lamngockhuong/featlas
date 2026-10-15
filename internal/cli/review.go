package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lamngockhuong/muninn/internal/content"
	"github.com/lamngockhuong/muninn/internal/review"
)

func newReviewCmd() *cobra.Command {
	var dir, by, note string
	var approve, requestChanges bool
	cmd := &cobra.Command{
		Use:   "review <kind>/<id> --approve|--request-changes --by <name> [--note <text>]",
		Short: "Approve or request changes on an entity, the same way the web does",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if approve == requestChanges {
				return errors.New("pass exactly one of --approve or --request-changes")
			}
			kindStr, id, ok := strings.Cut(args[0], "/")
			kind, kindOK := content.ParseKind(kindStr)
			if !ok || !kindOK || id == "" {
				return fmt.Errorf("entity must look like screen/claim-list, got %q", args[0])
			}
			l, err := load(dir)
			if err != nil {
				return err
			}
			e := l.ix.Get(kind, id)
			if e == nil {
				return fmt.Errorf("%s not found", args[0])
			}
			if errs := entityErrors(l.ix, e); len(errs) > 0 {
				for _, is := range errs {
					fmt.Fprintln(cmd.ErrOrStderr(), is.String())
				}
				return &exitError{code: 1, msg: fmt.Sprintf("%s has validation errors; fix them before reviewing", e.Key())}
			}
			d := review.Decision{Status: content.ReviewApproved, By: by, Note: note}
			if requestChanges {
				d.Status = content.ReviewChangesRequested
			}
			rv, err := review.Apply(l.ix, l.hasher, e, d)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s by %s\n", e.Key(), rv.Status, rv.By)
			return nil
		},
	}
	addDirFlag(cmd, &dir)
	cmd.Flags().BoolVar(&approve, "approve", false, "mark as approved")
	cmd.Flags().BoolVar(&requestChanges, "request-changes", false, "mark as changes requested (needs --note)")
	cmd.Flags().StringVar(&by, "by", "", "reviewer name (required)")
	cmd.Flags().StringVar(&note, "note", "", "review note")
	_ = cmd.MarkFlagRequired("by")
	return cmd
}

// entityErrors returns the error-severity issues of one entity.
func entityErrors(ix *content.Index, e *content.Entity) []content.Issue {
	var out []content.Issue
	for _, is := range ix.IssuesFor(e.Key()) {
		if is.Severity == content.SevError {
			out = append(out, is)
		}
	}
	return out
}
