package cmd

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/shire-studio/webhookhub-cli/internal/api"
	"github.com/shire-studio/webhookhub-cli/internal/auth"
	"github.com/spf13/cobra"
)

var endpointsCmd = &cobra.Command{
	Use:   "endpoints",
	Short: "List your endpoints (slug, name, public hook URL)",
	RunE: func(c *cobra.Command, _ []string) error {
		store, err := auth.NewStore()
		if err != nil {
			return err
		}
		return runEndpoints(apiBaseURL(), store, c.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(endpointsCmd)
}

func runEndpoints(baseURL string, store *auth.Store, out io.Writer) error {
	cfg, err := store.Load()
	if err != nil {
		return err
	}

	client := api.New(baseURL, cfg.Token)
	endpoints, err := client.Endpoints(context.Background())
	if err != nil {
		return fmt.Errorf("call /api/cli/endpoints: %w", err)
	}

	if len(endpoints) == 0 {
		fmt.Fprintln(out, "No endpoints. Create one at https://webhookhub.dev/endpoints.")
		return nil
	}

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SLUG\tNAME\tURL\t")
	for _, e := range endpoints {
		name := e.Name
		if !e.Active {
			name += " (inactive)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t\n", e.Slug, name, e.URL)
	}
	return tw.Flush()
}
