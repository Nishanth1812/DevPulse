package security

import (
	"regexp"
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
			regex:   regexp.MustCompile(`-----BEGIN\s+(?:RSA|DSA|EC|OPENSSH|PGP)\s+PRIVATE\s+KEY-----.+?-----END\s+(?:RSA|DSA|EC|OPENSSH|PGP)\s+PRIVATE\s+KEY-----`),
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
			name:    "generic-api-key",
			regex:   regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret|token|password)\s*[:=]\s*['"][^'"]+['"]`),
			replace: "[REDACTED CREDENTIAL]",
		},
	}
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

	return ScanResult{
		ContainsSecrets: len(matches) > 0,
		RedactedPrompt:  redacted,
		Matches:         matches,
	}
}
