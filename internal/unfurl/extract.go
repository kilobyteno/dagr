package unfurl

import (
	"net/url"
	"regexp"
	"strings"
)

const MaxURLsPerMessage = 3

var urlPattern = regexp.MustCompile(`https?://[^\s<>"'` + "`" + `]+`)

// ExtractURLs returns up to MaxURLsPerMessage unique http(s) URLs from body text.
func ExtractURLs(body string) []string {
	matches := urlPattern.FindAllString(body, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range matches {
		cleaned := strings.TrimRight(raw, ".,;:!?)]}>\"'")
		if cleaned == "" {
			continue
		}
		parsed, err := url.Parse(cleaned)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			continue
		}
		normalized := NormalizeURL(parsed)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, cleaned)
		if len(out) >= MaxURLsPerMessage {
			break
		}
	}
	return out
}

// NormalizeURL returns a stable key for deduplication.
func NormalizeURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	clone := *u
	clone.Fragment = ""
	clone.Scheme = strings.ToLower(clone.Scheme)
	clone.Host = strings.ToLower(clone.Host)
	if (clone.Scheme == "http" && strings.HasSuffix(clone.Host, ":80")) ||
		(clone.Scheme == "https" && strings.HasSuffix(clone.Host, ":443")) {
		host, _, _ := strings.Cut(clone.Host, ":")
		clone.Host = host
	}
	if clone.Path == "" {
		clone.Path = "/"
	}
	return clone.String()
}

func NormalizeURLString(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return NormalizeURL(parsed)
}
