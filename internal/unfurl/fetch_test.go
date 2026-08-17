package unfurl

import "testing"

func TestLooksLikeGIFURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		url  string
		want bool
	}{
		{"https://cdn.example/fun.gif", true},
		{"https://cdn.example/fun.GIF?x=1", true},
		{"https://media.giphy.com/media/abc123/giphy.gif", true},
		{"https://i.giphy.com/abc123.gif", true},
		{"https://media1.giphy.com/media/abc123/200.gif", true},
		{"https://media.tenor.com/xyz/tenor.gif", true},
		{"https://giphy.com/gifs/something-abc", false},
		{"https://example.com/page", false},
		{"https://cdn.example/img.png?format=gif", true},
	}
	for _, tc := range cases {
		if got := looksLikeGIFURL(tc.url); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.url, got, tc.want)
		}
	}
}

func TestParseHTMLMetaOpenGraph(t *testing.T) {
	t.Parallel()
	html := []byte(`<!doctype html><html><head>
<title>Fallback Title</title>
<meta property="og:title" content="OG Title" />
<meta property="og:description" content="OG Description" />
<meta property="og:site_name" content="Example Site" />
<meta property="og:image" content="https://cdn.example/img.png" />
</head><body></body></html>`)
	meta := parseHTMLMeta(html)
	if meta["og:title"] != "OG Title" {
		t.Fatalf("title=%q", meta["og:title"])
	}
	if meta["og:description"] != "OG Description" {
		t.Fatalf("description=%q", meta["og:description"])
	}
	if meta["og:site_name"] != "Example Site" {
		t.Fatalf("site=%q", meta["og:site_name"])
	}
	if meta["title"] != "Fallback Title" {
		t.Fatalf("fallback title=%q", meta["title"])
	}
	if meta["og:image"] != "https://cdn.example/img.png" {
		t.Fatalf("image=%q", meta["og:image"])
	}
}
