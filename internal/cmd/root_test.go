package cmd

import (
	"bytes"
	"testing"
)

func TestVersionVarsDefault(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version default = %q, want \"dev\"", Version)
	}
	if Commit != "none" {
		t.Errorf("Commit default = %q, want \"none\"", Commit)
	}
	if BuildDate != "unknown" {
		t.Errorf("BuildDate default = %q, want \"unknown\"", BuildDate)
	}
}

func TestVersionFlagOutput(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}

	got := buf.String()
	want := "webhookhub dev (commit none, built unknown)\n"
	if got != want {
		t.Errorf("--version output\n got: %q\nwant: %q", got, want)
	}
}
