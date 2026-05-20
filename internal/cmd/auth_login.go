package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shire-studio/webhookhub-cli/internal/api"
	"github.com/shire-studio/webhookhub-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate the CLI with a Personal Access Token",
	Long: `Prompts for a Personal Access Token (whk_…) created at
https://webhookhub.dev/profile/access-tokens and saves it to
$XDG_CONFIG_HOME/webhookhub/config.json (mode 0600).`,
	RunE: func(c *cobra.Command, _ []string) error {
		store, err := auth.NewStore()
		if err != nil {
			return err
		}
		return runAuthLogin(authLoginOpts{
			BaseURL:   apiBaseURL(),
			Store:     store,
			ReadToken: promptForTokenInteractive,
			Out:       c.OutOrStdout(),
		})
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
}

// authLoginOpts is split out so the command body is testable in isolation.
type authLoginOpts struct {
	BaseURL   string
	Store     *auth.Store
	ReadToken func() (string, error) // returns the user-entered token
	Out       io.Writer
}

func runAuthLogin(opts authLoginOpts) error {
	fmt.Fprintln(opts.Out, "Paste your token from https://webhookhub.dev/profile/access-tokens")
	fmt.Fprint(opts.Out, "Token: ")

	token, err := opts.ReadToken()
	if err != nil {
		return fmt.Errorf("read token: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("token cannot be empty")
	}

	client := api.New(opts.BaseURL, token)
	me, err := client.Me(context.Background())
	if err != nil {
		if errors.Is(err, api.ErrUnauthorized) {
			return errors.New("token rejected by server")
		}
		if errors.Is(err, api.ErrForbidden) {
			return errors.New("token does not have the 'cli' ability — recreate it with the cli ability checked")
		}
		return fmt.Errorf("validate token: %w", err)
	}

	if err := opts.Store.Save(auth.Config{Token: token}); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(opts.Out, "\n✓ Logged in as %s (%s plan).\n", me.Email, me.Plan)
	return nil
}

// promptForTokenInteractive reads a token from stdin. If stdin is a TTY,
// the input is hidden (term.ReadPassword); otherwise it falls back to
// reading the next line so scripted setups (`echo whk_… | webhookhub auth login`)
// work too.
func promptForTokenInteractive() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		raw, err := term.ReadPassword(fd)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
	var buf [4096]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil && err != io.EOF {
		return "", err
	}
	return string(buf[:n]), nil
}
