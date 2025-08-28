// cmd/api/main.go
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	scs "github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"

	"github.com/briangreenhill/coachgpt/internal/auth"
	"github.com/briangreenhill/coachgpt/internal/config"
	"github.com/briangreenhill/coachgpt/internal/db"
	"github.com/briangreenhill/coachgpt/internal/http/routes"
)

func main() {
	cfg := config.Load()

	// Logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	log.Printf("starting app on :%s", cfg.Port)

	// DB
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer pool.Close()
	queries := db.New(pool)

	// Sessions
	sess := scs.New()
	sess.Lifetime = 12 * time.Hour
	sess.Cookie.HttpOnly = true
	sess.Cookie.SameSite = http.SameSiteLaxMode
	sess.Cookie.Secure = false

	// Templates
	tmpl := template.Must(template.ParseGlob("web/templates/*.tmpl"))

	// Magic link helper
	ml := auth.MagicLink{
		Secret:  []byte(cfg.JWTSecret),
		BaseURL: cfg.BaseURL,
	}

	// Invite link helper
	inv := auth.InviteLink{
		Secret:  []byte(cfg.JWTSecret),
		BaseURL: cfg.BaseURL,
	}

	// Router / server
	s := routes.New(sess, tmpl, queries, ml, inv, cfg)
	h := hlog.NewHandler(logger)(s.Router)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: sess.LoadAndSave(h)}
	log.Fatal(srv.ListenAndServe())
}
