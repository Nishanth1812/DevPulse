package security

import (
	"strings"
	"testing"
)

func TestScanPrompt_GCPServiceAccount(t *testing.T) {
	input := `{
  "type": "service_account",
  "project_id": "my-project",
  "private_key_id": "abc123",
  "client_email": "sa@my-project.iam.gserviceaccount.com"
}`
	result := ScanPrompt(input)
	if !result.ContainsSecrets {
		t.Fatal("expected ContainsSecrets=true for GCP service account JSON")
	}
	found := false
	for _, m := range result.Matches {
		if m.Pattern == "gcp-service-account" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected gcp-service-account match")
	}
	if strings.Contains(result.RedactedPrompt, `"type": "service_account"`) {
		t.Fatal("GCP service account type field was not redacted")
	}
}

func TestScanPrompt_HighEntropyString(t *testing.T) {
	// High-entropy base64 string not preceded by a keyword pattern
	input := "session_key: dGhpcyBpcyBhIHJhbmRvbSB0b2tlbiB0aGF0IGlzIHZlcnkgbG9uZyBhbmQgc2VjcmV0"
	result := ScanPrompt(input)
	if !result.ContainsSecrets {
		t.Fatal("expected ContainsSecrets=true for high-entropy string")
	}
	found := false
	for _, m := range result.Matches {
		if m.Pattern == "high-entropy-string" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected high-entropy-string match, got matches=%v", result.Matches)
	}
}

func TestScanPrompt_LowEntropyStringNotFlagged(t *testing.T) {
	// Low entropy: repeated characters, English words
	input := "description = aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	result := ScanPrompt(input)
	for _, m := range result.Matches {
		if m.Pattern == "high-entropy-string" {
			t.Fatalf("low-entropy string was incorrectly flagged: %q", input)
		}
	}
}

func TestScanPrompt_AWSKey(t *testing.T) {
	input := "aws_key = AKIAIOSFODNN7EXAMPLE"
	result := ScanPrompt(input)
	if !result.ContainsSecrets {
		t.Fatal("expected ContainsSecrets=true for AWS key")
	}
	found := false
	for _, m := range result.Matches {
		if m.Pattern == "aws-access-key" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected aws-access-key match")
	}
}

func TestScanPrompt_GitHubToken(t *testing.T) {
	input := "export GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"
	result := ScanPrompt(input)
	if !result.ContainsSecrets {
		t.Fatal("expected ContainsSecrets=true for GitHub token")
	}
}

func TestScanPrompt_NoFalsePositiveOnNormalText(t *testing.T) {
	input := "The commit updates the README.md and adds unit tests for the parser module."
	result := ScanPrompt(input)
	if result.ContainsSecrets {
		t.Fatalf("normal text incorrectly flagged as containing secrets: matches=%v", result.Matches)
	}
}

func TestScanPrompt_ShannonEntropy(t *testing.T) {
	tests := []struct {
		input   string
		minEnt  float64
	}{
		{"AAAA", 0.0},
		{"ABCD", 2.0},
		{"dGhpcyBpcyBhIHJhbmRvbSB0b2tlbjQ=", 4.0},
	}
	for _, tt := range tests {
		e := shannonEntropy(tt.input)
		if e < tt.minEnt {
			t.Errorf("shannonEntropy(%q) = %f, want >= %f", tt.input, e, tt.minEnt)
		}
	}
}
