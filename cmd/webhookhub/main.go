// Command webhookhub is the official CLI for WebhookHub.
//
// See https://webhookhub.dev for product docs.
package main

import (
	"fmt"
	"os"

	"github.com/shire-studio/webhookhub-cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
