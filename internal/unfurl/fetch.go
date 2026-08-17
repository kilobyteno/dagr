package unfurl

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const (
	maxRedirects  = 3
	maxBodyBytes  = 1 << 20 // 1 MiB
	fetchTimeout  = 5 * time.Second
	maxTitleRunes = 200
	maxDescRunes  = 400
	maxSiteRunes  = 120
	maxImageRunes = 2048
)

// Result is metadata extracted from a remote page.
type Result struct {
	URL         string
	Title       string
	Description string
	SiteName    string
	ImageURL    string
}

// Fetch loads URL metadata with SSRF protections.
func Fetch(ctx context.Context, rawURL string) (Result, error) {
	parsed, err := validateFetchURL(rawURL)
	if err != nil {
		return Result{}, err
	}

	client := &http.Client{
		Timeout: fetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects")
			}
			if _, err := validateFetchURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   3 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   3 * time.Second,
			ResponseHeaderTimeout: 3 * time.Second,
			ForceAttemptHTTP2:     true,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return Result{}, err
	}
	// Browser-like UA: many sites serve empty shells or block unknown bot agents.
	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (compatible; DagrLinkPreview/1.0; +https://dagr.no) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,image/gif;q=0.8,image/*;q=0.7,*/*;q=0.5")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	base, _ := url.Parse(finalURL)
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	mediaType, _, _ := strings.Cut(contentType, ";")
	mediaType = strings.TrimSpace(mediaType)
	isHTML := strings.Contains(mediaType, "text/html") ||
		strings.Contains(mediaType, "application/xhtml")
	isImage := strings.HasPrefix(mediaType, "image/")
	// Direct image responses, or GIF-like URLs when the server omits a useful type.
	if isImage || (!isHTML && looksLikeGIFURL(finalURL)) {
		// Direct media (especially linked GIFs): do not parse as HTML.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		site := hostLabel(base)
		image := truncateRunes(strings.TrimSpace(finalURL), maxImageRunes)
		if _, err := validateFetchURL(image); err != nil {
			return Result{}, err
		}
		return Result{
			URL:      finalURL,
			Title:    site,
			SiteName: site,
			ImageURL: image,
		}, nil
	}
	if mediaType != "" && !isHTML {
		return Result{}, fmt.Errorf("unsupported content type")
	}

	limited := io.LimitReader(resp.Body, maxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return Result{}, err
	}
	if len(body) > maxBodyBytes {
		body = body[:maxBodyBytes]
	}

	meta := parseHTMLMeta(body)

	title := firstNonEmpty(meta["og:title"], meta["twitter:title"], meta["title"])
	desc := firstNonEmpty(meta["og:description"], meta["twitter:description"], meta["description"])
	site := firstNonEmpty(meta["og:site_name"], meta["application-name"], hostLabel(base))
	image := firstNonEmpty(meta["og:image"], meta["twitter:image"], meta["twitter:image:src"])
	if image != "" && base != nil {
		if abs, err := base.Parse(image); err == nil {
			image = abs.String()
		}
	}
	if image != "" {
		if _, err := validateFetchURL(image); err != nil {
			image = ""
		}
	}

	return Result{
		URL:         finalURL,
		Title:       truncateRunes(sanitizeText(title), maxTitleRunes),
		Description: truncateRunes(sanitizeText(desc), maxDescRunes),
		SiteName:    truncateRunes(sanitizeText(site), maxSiteRunes),
		ImageURL:    truncateRunes(strings.TrimSpace(image), maxImageRunes),
	}, nil
}

func parseHTMLMeta(body []byte) map[string]string {
	out := map[string]string{}
	tokenizer := html.NewTokenizer(strings.NewReader(string(body)))
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return out
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			switch token.Data {
			case "title":
				if tokenizer.Next() == html.TextToken {
					if _, ok := out["title"]; !ok {
						out["title"] = string(tokenizer.Text())
					}
				}
			case "meta":
				var prop, name, content string
				for _, attr := range token.Attr {
					switch strings.ToLower(attr.Key) {
					case "property":
						prop = strings.ToLower(strings.TrimSpace(attr.Val))
					case "name":
						name = strings.ToLower(strings.TrimSpace(attr.Val))
					case "content":
						content = strings.TrimSpace(attr.Val)
					}
				}
				if content == "" {
					continue
				}
				key := firstNonEmpty(prop, name)
				if key == "" {
					continue
				}
				if _, ok := out[key]; !ok {
					out[key] = content
				}
			case "body":
				return out
			}
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func sanitizeText(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}

func hostLabel(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Hostname()
}

// looksLikeGIFURL is a best-effort check for direct GIF media URLs.
func looksLikeGIFURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.Contains(strings.ToLower(raw), ".gif")
	}
	path := strings.ToLower(parsed.Path)
	if strings.HasSuffix(path, ".gif") || strings.Contains(path, ".gif/") {
		return true
	}
	q := parsed.Query()
	if strings.EqualFold(q.Get("format"), "gif") || strings.EqualFold(q.Get("fm"), "gif") {
		return true
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "i.giphy.com" || (isDomainOrSubdomain(host, "giphy.com") && strings.HasPrefix(host, "media")) {
		return true
	}
	if isDomainOrSubdomain(host, "tenor.com") && strings.Contains(path, "gif") {
		return true
	}
	return false
}

func isDomainOrSubdomain(host, root string) bool {
	return host == root || strings.HasSuffix(host, "."+root)
}
