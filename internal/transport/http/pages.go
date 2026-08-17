package http

import (
	"html/template"
	"net/http"
	"strings"
)

type publicPage struct {
	Title string
	Body  string
	Token string
}

var publicPageTmpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} · Dagr</title>
<style>
  :root { color-scheme: light dark; }
  body {
    font-family: ui-sans-serif, system-ui, sans-serif;
    margin: 0;
    min-height: 100vh;
    display: grid;
    place-items: center;
    background: #0f1115;
    color: #e8eaed;
  }
  main {
    width: min(28rem, calc(100% - 2rem));
    padding: 1.75rem;
    border: 1px solid #2a2f3a;
    border-radius: 12px;
    background: #171a21;
  }
  h1 { font-size: 1.25rem; margin: 0 0 0.75rem; }
  p { margin: 0; line-height: 1.5; color: #c4c8d0; }
  button {
    margin-top: 1.25rem;
    appearance: none;
    border: 0;
    border-radius: 8px;
    padding: 0.65rem 1rem;
    font: inherit;
    font-weight: 600;
    background: #5b8def;
    color: #fff;
    cursor: pointer;
  }
</style>
</head>
<body>
<main>
  <h1>{{.Title}}</h1>
  <p>{{.Body}}</p>
  {{if .Token}}
  <form method="post" action="/verify-email" id="verify">
    <input type="hidden" name="token" value="{{.Token}}">
    <button type="submit">Verify email</button>
  </form>
  <script>document.getElementById("verify").submit()</script>
  {{end}}
</main>
</body>
</html>`))

func writePublicPage(w http.ResponseWriter, status int, page publicPage) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = publicPageTmpl.Execute(w, page)
}

func (s *Server) handleVerifyEmailPage(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		writePublicPage(w, http.StatusBadRequest, publicPage{
			Title: "Verify email",
			Body:  "This link is missing a token. Request a new verification message from Dagr.",
		})
		return
	}
	writePublicPage(w, http.StatusOK, publicPage{
		Title: "Verify email",
		Body:  "Confirm this email address for your Dagr account.",
		Token: token,
	})
}

func (s *Server) handleVerifyEmailForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writePublicPage(w, http.StatusBadRequest, publicPage{
			Title: "Verify email",
			Body:  "This link could not be read. Request a new verification message from Dagr.",
		})
		return
	}
	user, err := s.auth.VerifyEmail(r.Context(), r.FormValue("token"))
	if err != nil {
		writePublicPage(w, http.StatusBadRequest, publicPage{
			Title: "Verify email",
			Body:  "This link is invalid or has expired. Request a new verification message from Dagr.",
		})
		return
	}
	writePublicPage(w, http.StatusOK, publicPage{
		Title: "Email verified",
		Body:  "Thanks, " + user.DisplayName + ". You can return to Dagr.",
	})
}

func (s *Server) handleAcceptInvitePage(w http.ResponseWriter, r *http.Request) {
	writePublicPage(w, http.StatusOK, publicPage{
		Title: "Workspace invite",
		Body:  "Open the Dagr app and sign in to accept this invite. If you are already signed in, ask the person who invited you to send it again from the app.",
	})
}

var notFoundPageTmpl = template.Must(template.New("notfound").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>404 · Dagr</title>
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    min-height: 100vh;
    display: grid;
    place-items: center;
    font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
    background:
      radial-gradient(900px 480px at 50% 0%, rgba(242, 103, 34, 0.18), transparent 58%),
      #121212;
    color: #f6f6f6;
  }
  .wrap { text-align: center; padding: 2rem 1.25rem; }
  .brand {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 2.5rem;
    font-weight: 600;
    letter-spacing: -0.02em;
  }
  .dot {
    width: 0.65rem;
    height: 0.65rem;
    border-radius: 999px;
    background: #f26722;
    box-shadow: 0 0 16px rgba(242, 103, 34, 0.7);
  }
  .code {
    margin: 0;
    font-size: clamp(5.5rem, 18vw, 9rem);
    font-weight: 700;
    letter-spacing: -0.08em;
    line-height: 0.9;
    background: linear-gradient(180deg, #fff 18%, #f26722 150%);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
  }
  h1 {
    margin: 1.25rem 0 0.6rem;
    font-size: 1.35rem;
    font-weight: 600;
  }
  p {
    margin: 0 auto;
    max-width: 22rem;
    line-height: 1.55;
    color: #b4b4b4;
  }
</style>
</head>
<body>
  <div class="wrap">
    <div class="brand"><span class="dot"></span> Dagr</div>
    <p class="code">404</p>
    <h1>This page is not here</h1>
    <p>Dagr has no page at this address. Open the app to continue.</p>
  </div>
</body>
</html>`))

func writeNotFoundPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	_ = notFoundPageTmpl.Execute(w, nil)
}

func isWebVisitor(r *http.Request) bool {
	path := r.URL.Path
	if path == "/api" || strings.HasPrefix(path, "/api/") {
		return false
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	if accept == "" {
		return false
	}
	htmlIdx := strings.Index(accept, "text/html")
	if htmlIdx < 0 {
		return false
	}
	jsonIdx := strings.Index(accept, "application/json")
	if jsonIdx >= 0 && jsonIdx < htmlIdx {
		return false
	}
	return true
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if !isWebVisitor(r) {
		s.writeError(w, r, http.StatusNotFound, "not_found", "Not found", nil)
		return
	}
	writeNotFoundPage(w)
}
