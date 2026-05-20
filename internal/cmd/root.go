// Package cmd hosts the Cobra command tree for the webhookhub CLI.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version, Commit, BuildDate are injected at build time by GoReleaser via -ldflags.
// See .goreleaser.yaml.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

const defaultAPIURL = "https://webhookhub.dev"

// rootCmd is the entry point for the Cobra tree.
var rootCmd = &cobra.Command{
	Use:           "webhookhub",
	Short:         "Tunnel webhooks captured at a stable WebhookHub URL down to your local server",
	Version:       fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, BuildDate),
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.SetVersionTemplate("{{.Use}} {{.Version}}\n")
}

// Execute runs the root command. Called by cmd/webhookhub/main.go.
func Execute() error {
	return rootCmd.Execute()
}

// apiBaseURL returns the API base URL — env override > default.
func apiBaseURL() string {
	if v := os.Getenv("WEBHOOKHUB_API_URL"); v != "" {
		return v
	}
	return defaultAPIURL
}
