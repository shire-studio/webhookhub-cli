package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/shire-studio/webhookhub-cli/internal/api"
	"github.com/shire-studio/webhookhub-cli/internal/auth"
	"github.com/shire-studio/webhookhub-cli/internal/forward"
	"github.com/spf13/cobra"
)

var (
	forwardTimeout time.Duration
	forwardVerbose bool
)

var forwardCmd = &cobra.Command{
	Use:   "forward [slug]",
	Short: "Tunnel webhooks to a local server",
	Long: `Stream every webhook hitting the given endpoint slug down to your local
development server.

Single-endpoint mode:
  webhookhub forward stripe-test --to http://localhost:3000

Multi-endpoint mode (reads webhookhub.yaml or
$XDG_CONFIG_HOME/webhookhub/forwards.yaml):
  webhookhub forward`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		store, err := auth.NewStore()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(c.Context())
		defer cancel()

		if len(args) == 0 {
			// Multi-endpoint mode (Task 8 fills this in)
			return runForwardMulti(forwardOpts{
				BaseURL: apiBaseURL(),
				Store:   store,
				Timeout: forwardTimeout,
				Verbose: forwardVerbose,
				Out:     c.OutOrStdout(),
				Ctx:     ctx,
			})
		}

		to, _ := c.Flags().GetString("to")
		if to == "" {
			return errors.New("--to is required when a slug is given")
		}

		return runForwardSingle(forwardOpts{
			BaseURL:   apiBaseURL(),
			Store:     store,
			Slug:      args[0],
			TargetURL: to,
			Timeout:   forwardTimeout,
			Verbose:   forwardVerbose,
			Out:       c.OutOrStdout(),
			Ctx:       ctx,
		})
	},
}

func init() {
	forwardCmd.Flags().String("to", "", "Local URL to replay webhooks against (single-endpoint mode)")
	forwardCmd.Flags().DurationVar(&forwardTimeout, "timeout", 30*time.Second, "Per-replay request timeout")
	forwardCmd.Flags().BoolVar(&forwardVerbose, "verbose", false, "Log request/response headers and bodies")
	rootCmd.AddCommand(forwardCmd)
}

// forwardOpts is the testable seam for both modes.
type forwardOpts struct {
	BaseURL   string
	Store     *auth.Store
	Slug      string            // single-endpoint mode
	TargetURL string            // single-endpoint mode
	Mappings  map[string]string // multi-endpoint mode (slug → targetURL)
	Timeout   time.Duration
	Verbose   bool
	Out       io.Writer
	Ctx       context.Context
}

// runForwardSingle handles `webhookhub forward <slug> --to <url>`.
func runForwardSingle(opts forwardOpts) error {
	cfg, err := opts.Store.Load()
	if err != nil {
		return err
	}

	target, err := url.Parse(opts.TargetURL)
	if err != nil {
		return fmt.Errorf("parse --to URL: %w", err)
	}

	client := api.New(opts.BaseURL, cfg.Token)

	// Validate the slug exists for this user.
	endpoints, err := client.Endpoints(opts.Ctx)
	if err != nil {
		if errors.Is(err, api.ErrUnauthorized) {
			return errors.New("token rejected by server (run `webhookhub auth login`)")
		}
		return fmt.Errorf("list endpoints: %w", err)
	}
	known := false
	for _, e := range endpoints {
		if e.Slug == opts.Slug {
			known = true
			break
		}
	}
	if !known {
		return fmt.Errorf("endpoint slug %q not found in your account", opts.Slug)
	}

	mappings := map[string]*url.URL{opts.Slug: target}
	return streamLoop(opts.Ctx, client, []string{opts.Slug}, mappings, opts)
}

// runForwardMulti is implemented in Task 8.
func runForwardMulti(opts forwardOpts) error {
	return errors.New("multi-endpoint mode is implemented in Task 8")
}

// streamLoop is the reconnect-with-backoff wrapper around api.Client.Stream.
// Returns when ctx cancels, the loop hits a terminal error (401/403), or
// the slug filter resolves to no valid endpoints.
func streamLoop(
	ctx context.Context,
	client *api.Client,
	slugs []string,
	mappings map[string]*url.URL,
	opts forwardOpts,
) error {
	logger := forward.NewLogger(opts.Out, opts.Verbose)

	fmt.Fprintf(opts.Out, "✓ Connected to %s\n", opts.BaseURL)
	for slug, t := range mappings {
		fmt.Fprintf(opts.Out, "✓ Tunneling %s → %s\n", slug, t.String())
	}

	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second}
	idx := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		err := client.Stream(ctx, slugs, api.StreamHandlers{
			OnConnected: func(api.ConnectedPayload) {
				idx = 0 // reset backoff on successful (re)connect
			},
			OnWebhook: func(ev api.WebhookEvent) {
				idx = 0 // reset backoff on any successful frame too
				prefix := ""
				if len(mappings) > 1 {
					prefix = ev.EndpointSlug
				}
				target, ok := mappings[ev.EndpointSlug]
				if !ok {
					return // event for an endpoint we aren't tunneling
				}

				res := forward.Replay(ctx, target, ev, opts.Timeout)
				logger.LogReplay(prefix, ev, target, res)

				lr := api.LocalResponseInput{
					Status:     res.Status,
					Headers:    res.Headers,
					Body:       res.Body,
					DurationMs: res.DurationMs,
					Error:      res.ErrorCode,
				}
				if pbErr := client.PostLocalResponse(ctx, ev.RequestID, lr); pbErr != nil {
					if opts.Verbose {
						fmt.Fprintf(opts.Out, "    post-back failed: %v\n", pbErr)
					}
				}
			},
			OnPing: func() { /* heartbeat — nothing to do */ },
		})

		if errors.Is(err, api.ErrUnauthorized) {
			return errors.New("token rejected by server (run `webhookhub auth login`)")
		}
		if errors.Is(err, api.ErrForbidden) {
			return errors.New("token does not have the 'cli' ability — recreate it with the cli ability checked")
		}
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return nil
		}

		// Network/server failure — back off and retry
		wait := backoffs[idx]
		if idx < len(backoffs)-1 {
			idx++
		}
		fmt.Fprintf(opts.Out, "✗ Stream ended (%v); reconnecting in %s\n", err, wait)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
		}
	}
}
