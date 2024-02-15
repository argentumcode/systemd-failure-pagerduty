package main

import "testing"

func TestGenerateSummary(t *testing.T) {
	summary := generateSummary(DEFAULT_SUMMARY_TEMPLATE, "host", "unit")
	if summary != "[host] systemd failure unit" {
		t.Errorf("generateSummary() = \"%s\"; want \"[host] systemd failure unit\"", summary)
	}
}

func TestGenerateSummaryError(t *testing.T) {
	summary := generateSummary("{{.AAAA}}", "host", "unit")
	if summary != "[host] systemd failure unit" {
		t.Errorf("generateSummary() = \"%s\"; want \"[host] systemd failure unit\"", summary)
	}
}
