// cmd/api/main.go
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	scs "github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"

	"github.com/briangreenhill/coachgpt/internal/auth"
	"github.com/briangreenhill/coachgpt/internal/config"
	"github.com/briangreenhill/coachgpt/internal/db"
	"github.com/briangreenhill/coachgpt/internal/email"
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

	// Templates with custom functions
	funcMap := template.FuncMap{
		"div":  func(a, b int32) int32 { return a / b },
		"mod":  func(a, b int32) int32 { return a % b },
		"divf": func(a, b float64) float64 { return a / b },
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/*.tmpl"))

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

	// Mail sender (MailHog on localhost:1025)
	sender := email.NewSMTPSender("localhost:1025", "no-reply@coachgpt.local")

	// Router / server
	s := routes.New(routes.ServerOptions{
		Sess:   sess,
		Tmpl:   tmpl,
		Q:      queries,
		Magic:  ml,
		Invite: inv,
		Cfg:    cfg,
		Email:  sender,
	})
	h := hlog.NewHandler(logger)(s.Router)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: sess.LoadAndSave(h)}
	log.Printf("starting app on :%s", cfg.Port)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)

		// Create a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	case err := <-serverErr:
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Server shutdown complete")
}
