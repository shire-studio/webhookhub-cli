package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/shire-studio/webhookhub-cli/internal/api"
	"github.com/shire-studio/webhookhub-cli/internal/auth"
	"github.com/spf13/cobra"
)

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the email and plan of the active token",
	RunE: func(c *cobra.Command, _ []string) error {
		store, err := auth.NewStore()
		if err != nil {
			return err
		}
		return runAuthWhoami(apiBaseURL(), store, c.OutOrStdout())
	},
}

func init() {
	authCmd.AddCommand(authWhoamiCmd)
}

func runAuthWhoami(baseURL string, store *auth.Store, out io.Writer) error {
	cfg, err := store.Load()
	if err != nil {
		return err // ErrNotLoggedIn pretty-prints itself
	}

	client := api.New(baseURL, cfg.Token)
	me, err := client.Me(context.Background())
	if err != nil {
		if errors.Is(err, api.ErrUnauthorized) {
			return errors.New("token rejected by server (was it revoked? run `webhookhub auth login`)")
		}
		return fmt.Errorf("call /api/cli/me: %w", err)
	}

	fmt.Fprintf(out, "%s (%s plan)\n", me.Email, me.Plan)
	return nil
}
