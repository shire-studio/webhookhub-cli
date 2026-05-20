package cmd

import (
	"fmt"
	"io"

	"github.com/shire-studio/webhookhub-cli/internal/auth"
	"github.com/spf13/cobra"
)

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Delete the saved auth token",
	RunE: func(c *cobra.Command, _ []string) error {
		store, err := auth.NewStore()
		if err != nil {
			return err
		}
		return runAuthLogout(store, c.OutOrStdout())
	},
}

func init() {
	authCmd.AddCommand(authLogoutCmd)
}

func runAuthLogout(store *auth.Store, out io.Writer) error {
	if err := store.Delete(); err != nil {
		return fmt.Errorf("delete config: %w", err)
	}
	fmt.Fprintln(out, "✓ Logged out.")
	return nil
}
