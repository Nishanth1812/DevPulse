package security

import (
	"math"
	"regexp"
	"strings"
)

type Match struct {
	Pattern string
}

type ScanResult struct {
	ContainsSecrets bool
	RedactedPrompt  string
	Matches         []Match
}

type pattern struct {
	name    string
	regex   *regexp.Regexp
	replace string
}

var patterns []pattern

func init() {
	patterns = []pattern{
		{
			name:    "pem-private-key",
			regex:   regexp.MustCompile(`-----BEGIN\s+(?:RSA|DSA|EC|OPENSSH|PGP)\s+PRIVATE\s+KEY-----[\s\S]+?-----END\s+(?:RSA|DSA|EC|OPENSSH|PGP)\s+PRIVATE\s+KEY-----`),
			replace: "[REDACTED PEM PRIVATE KEY]",
		},
		{
			name:    "github-token",
			regex:   regexp.MustCompile(`(?:ghp|gho|ghu|ghs|ghr)_[0-9a-zA-Z]{36,}`),
			replace: "[REDACTED GITHUB TOKEN]",
		},
		{
			name:    "api-key-sk",
			regex:   regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),
			replace: "[REDACTED API KEY]",
		},
		{
			name:    "aws-access-key",
			regex:   regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			replace: "[REDACTED AWS KEY]",
		},
		{
			name:    "jwt",
			regex:   regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{10,}\.(?:[a-zA-Z0-9_-]{10,}\.){1,2}[a-zA-Z0-9_-]{10,}`),
			replace: "[REDACTED JWT]",
		},
		{
			name:    "slack-token",
			regex:   regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z-]{10,}`),
			replace: "[REDACTED SLACK TOKEN]",
		},
		{
			name:    "gcp-service-account",
			regex:   regexp.MustCompile(`"type"\s*:\s*"service_account"`),
			replace: "[REDACTED GCP SERVICE ACCOUNT]",
		},
		{
			name:    "generic-api-key",
			regex:   regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret|token|password)\s*[:=]\s*['"][^'"]+['"]`),
			replace: "[REDACTED CREDENTIAL]",
		},
		{
			name:    "generic-api-key-unquoted",
			regex:   regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret|token|password|passwd)\s*[:=]\s*\S+`),
			replace: "[REDACTED CREDENTIAL]",
		},
		{
			name:    "connection-string",
			regex:   regexp.MustCompile(`(?i)[a-z][a-z0-9+.\-]*://[^\s:@/]+:[^\s@/]+@`),
			replace: "[REDACTED CONNECTION STRING]",
		},
	}
}

var highEntropyRe = regexp.MustCompile(`[A-Za-z0-9+/=]{40,}`)

// highEntropyHintRe matches a secret-associated keyword on the same line before
// a candidate, so plain long base64 blobs (data URIs, image data, tokens in
// diffs) are not flagged as secrets.
var highEntropyHintRe = regexp.MustCompile(`(?i)(key|secret|token|password|passwd|credential|bearer|session|authorization)\b`)

func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	freq := make(map[rune]float64)
	for _, c := range s {
		freq[c]++
	}
	length := float64(len([]rune(s)))
	entropy := 0.0
	for _, count := range freq {
		p := count / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

func ScanPrompt(prompt string) ScanResult {
	redacted := prompt
	var matches []Match

	for _, p := range patterns {
		locs := p.regex.FindAllStringIndex(redacted, -1)
		if locs == nil {
			continue
		}
		for range locs {
			matches = append(matches, Match{Pattern: p.name})
		}
		redacted = p.regex.ReplaceAllString(redacted, p.replace)
	}

	for _, line := range strings.Split(redacted, "\n") {
		// Only consider a high-entropy run a secret when a hint keyword appears
		// on the same line before it. This keeps data URIs and long base64
		// payloads (images, binary blobs) from being over-flagged.
		if !highEntropyHintRe.MatchString(line) {
			continue
		}
		for _, c := range highEntropyRe.FindAllString(line, -1) {
			if shannonEntropy(c) <= 4.5 {
				continue
			}
			matches = append(matches, Match{Pattern: "high-entropy-string"})
			redacted = strings.ReplaceAll(redacted, c, "[REDACTED HIGH-ENTROPY STRING]")
		}
	}

	return ScanResult{
		ContainsSecrets: len(matches) > 0,
		RedactedPrompt:  redacted,
		Matches:         matches,
	}
}
