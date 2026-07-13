// Package redaction redacts secrets and PII from model-visible and Lab-facing
// text (Chunk 30). Raw corpus/index text is not rewritten.
package redaction

import (
	"os"
	"regexp"
	"strings"
)

const (
	// Replacement is the opaque stand-in for redacted spans.
	Replacement = "[REDACTED]"
)

// Report summarizes how many replacements were applied.
type Report struct {
	Applied bool `json:"applied"`
	Count   int  `json:"count"`
}

// Redactor transforms text for model/Lab-visible surfaces.
type Redactor interface {
	Redact(s string) (string, Report)
}

// Default applies deterministic stdlib patterns (no vendor DLP).
type Default struct{}

var (
	reBearer = regexp.MustCompile(`(?i)\b(bearer\s+)([a-z0-9._\-+=/]{8,})`)
	// No leading \b: localecho %q may emit \napi_key=… where n is a word char.
	reAPIKey = regexp.MustCompile(`(?i)(api[_-]?key|access[_-]?token|secret[_-]?key|auth[_-]?token)(\s*[=:]\s*)([^\s"'\\]{8,})`)
	reAssign = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_])(password|passwd|token|secret)(\s*[=:]\s*)([^\s"'\\]{4,})`)
	reEmail  = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)
)

func (Default) Redact(s string) (string, Report) {
	if s == "" {
		return s, Report{}
	}
	out := s
	n := 0
	replace := func(re *regexp.Regexp, keepPrefixGroups int) {
		out = re.ReplaceAllStringFunc(out, func(m string) string {
			n++
			if keepPrefixGroups <= 0 {
				return Replacement
			}
			sub := re.FindStringSubmatch(m)
			if len(sub) <= keepPrefixGroups {
				return Replacement
			}
			var b strings.Builder
			for i := 1; i <= keepPrefixGroups; i++ {
				b.WriteString(sub[i])
			}
			b.WriteString(Replacement)
			return b.String()
		})
	}
	replace(reBearer, 1)
	replace(reAPIKey, 2)
	replace(reAssign, 3)
	replace(reEmail, 0)
	return out, Report{Applied: n > 0, Count: n}
}

// Enabled reports whether CONTEXT_REDACT is on (default true).
// Set CONTEXT_REDACT=0|false|off to disable.
func Enabled() bool {
	v := strings.TrimSpace(os.Getenv("CONTEXT_REDACT"))
	if v == "" {
		return true
	}
	return !(v == "0" || strings.EqualFold(v, "false") || strings.EqualFold(v, "off"))
}

// Apply uses Default when Enabled; otherwise returns s unchanged.
func Apply(s string) (string, Report) {
	if !Enabled() {
		return s, Report{}
	}
	return (Default{}).Redact(s)
}
