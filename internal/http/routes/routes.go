package routes

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	scs "github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/briangreenhill/coachgpt/internal/auth"
	"github.com/briangreenhill/coachgpt/internal/db"
	appmw "github.com/briangreenhill/coachgpt/internal/http/middleware"
)

type Server struct {
	Router  *chi.Mux
	Sess    *scs.SessionManager
	Tmpl    *template.Template
	Q       *db.Queries    // sqlc queries
	Magic   auth.MagicLink // magic-link helper
	BaseURL string
}

func New(sess *scs.SessionManager, tmpl *template.Template, queries *db.Queries, ml auth.MagicLink, baseURL string) *Server {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	s := &Server{Router: r, Sess: sess, Tmpl: tmpl, Q: queries, Magic: ml, BaseURL: baseURL}

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	r.Get("/", s.handleHome)
	r.Get("/login", s.handleLogin)
	r.Post("/auth/magic-link", s.handleMagicLink)
	r.Get("/auth/callback", s.handleCallback)

	r.Group(func(pr chi.Router) {
		pr.Use(s.sessionToContext)
		pr.Use(appmw.RequireAuth)
		pr.Get("/dashboard", s.handleDashboard)
	})

	return s
}

func (s *Server) sessionToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := s.Sess.GetString(r.Context(), "coach_id"); id != "" {
			// use the SAME key that RequireAuth checks
			r = r.WithContext(context.WithValue(r.Context(), appmw.CoachIDKey, id))
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.Tmpl.ExecuteTemplate(w, name, data)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	s.render(w, "login", map[string]any{"Title": "Login"})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.render(w, "dashboard", map[string]any{"Title": "Dashboard"})
}

func (s *Server) handleMagicLink(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", 400)
		return
	}
	email := strings.TrimSpace(r.Form.Get("email"))
	if email == "" {
		http.Error(w, "email required", 400)
		return
	}

	url, err := s.issueMagicLink(r.Context(), email)
	if err != nil {
		http.Error(w, "could not issue link", 500)
		return
	}

	log.Printf("[auth] magic link for %s: %s", email, url)
	s.render(w, "magic_sent", map[string]any{
		"Title": "Magic Link Sent", "Email": email, "URL": url,
	})
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// ---- Magic link flow

func (s *Server) issueMagicLink(ctx context.Context, email string) (string, error) {
	// Upsert coach so callback can find them
	_, err := s.Q.UpsertCoachByEmail(ctx, db.UpsertCoachByEmailParams{
		Email: email,
		Name:  pgtype.Text{String: "", Valid: false},
		Tz:    "Europe/Berlin",
	})
	if err != nil {
		return "", err
	}
	return s.Magic.URL(email, 2*time.Hour), nil // long TTL while developing
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	if un, err := url.QueryUnescape(tok); err == nil {
		tok = un
	}

	email, err := s.Magic.Verify(tok)
	if err != nil {
		log.Printf("[auth] verify failed: %v", err)
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	coach, err := s.Q.GetCoachByEmail(r.Context(), email)
	if err != nil {
		log.Printf("[auth] coach lookup failed for %s: %v", email, err)
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	s.Sess.Put(r.Context(), "coach_id", coach.ID.String())
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}
