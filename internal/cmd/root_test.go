package cmd

import (
	"strings"
	"testing"
)

func TestVersionVarsDefault(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version default = %q, want \"dev\"", Version)
	}
	if Commit != "none" {
		t.Errorf("Commit default = %q, want \"none\"", Commit)
	}
	if Date != "unknown" {
		t.Errorf("Date default = %q, want \"unknown\"", Date)
	}
}

func TestVersionTemplateReferencesAllThreeVars(t *testing.T) {
	tmpl := rootCmd.VersionTemplate()
	if tmpl == "" {
		t.Fatal("VersionTemplate is empty")
	}
	for _, s := range []string{".Version", Commit, Date} {
		if !strings.Contains(tmpl, s) {
			t.Errorf("VersionTemplate missing %q substring: %s", s, tmpl)
		}
	}
}
