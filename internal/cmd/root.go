// Package cmd hosts the Cobra command tree for the webhookhub CLI.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Version, Commit, Date are injected at build time by GoReleaser via -ldflags.
// See .goreleaser.yaml.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

const defaultAPIURL = "https://webhookhub.dev"

// rootCmd is the entry point for the Cobra tree.
var rootCmd = &cobra.Command{
	Use:           "webhookhub",
	Short:         "Tunnel webhooks captured at a stable WebhookHub URL down to your local server",
	Version:       Version,
	SilenceUsage:  true, // don't dump usage on every error
	SilenceErrors: true, // we print to stderr ourselves in main()
}

func init() {
	rootCmd.SetVersionTemplate(
		`webhookhub {{.Version}} (commit ` + Commit + `, built ` + Date + `)` + "\n",
	)
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
