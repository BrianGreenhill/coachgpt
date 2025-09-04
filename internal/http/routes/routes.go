package routes

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	scs "github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"

	"github.com/briangreenhill/coachgpt/internal/auth"
	"github.com/briangreenhill/coachgpt/internal/config"
	"github.com/briangreenhill/coachgpt/internal/db"
	"github.com/briangreenhill/coachgpt/internal/email"
	appmw "github.com/briangreenhill/coachgpt/internal/http/middleware"
	"github.com/briangreenhill/coachgpt/internal/jobs"
)

type Server struct {
	Router      *chi.Mux
	Sess        *scs.SessionManager
	Tmpl        *template.Template
	Q           *db.Queries    // sqlc queries
	Magic       auth.MagicLink // magic-link helper
	BaseURL     string
	Invite      auth.InviteLink // invite-link helper
	StravaConf  *oauth2.Config
	StateSecret string // for signing oauth2 state param
	RedisAddr   string
	Email       email.Sender
}

type ServerOptions struct {
	Sess   *scs.SessionManager
	Tmpl   *template.Template
	Q      *db.Queries
	Magic  auth.MagicLink
	Invite auth.InviteLink
	Cfg    config.Config
	Email  email.Sender
}

func New(opts ServerOptions) *Server {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	s := &Server{Router: r, Sess: opts.Sess, Tmpl: opts.Tmpl, Q: opts.Q, Magic: opts.Magic, BaseURL: opts.Cfg.BaseURL, Invite: opts.Invite, StateSecret: opts.Cfg.JWTSecret, RedisAddr: opts.Cfg.RedisAddr, Email: opts.Email}
	s.StravaConf = &oauth2.Config{
		ClientID:     opts.Cfg.Strava.ClientID,
		ClientSecret: opts.Cfg.Strava.ClientSecret,
		RedirectURL:  opts.Cfg.BaseURL + "/oauth/strava/callback",
		Scopes:       []string{"read", "activity:read_all"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.strava.com/oauth/authorize",
			TokenURL: "https://www.strava.com/oauth/token",
		},
	}

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("Error writing health check response: %v", err)
		}
	})

	r.Get("/", s.handleHome)
	r.Get("/login", s.handleLogin)
	r.Post("/auth/magic-link", s.handleMagicLink)
	r.Get("/auth/callback", s.handleCallback)
	r.Get("/invite", s.handleAthleteInvite) // public, but needs token
	r.Get("/oauth/strava/start", s.handleStravaStart)
	r.Get("/oauth/strava/callback", s.handleStravaCallback)
	r.Post("/interest", s.handleInterestSubmit)

	r.Group(func(pr chi.Router) {
		pr.Use(s.sessionToContext)
		pr.Use(appmw.RequireAuth)
		pr.Post("/athletes", s.handleCreateAthlete)
		pr.Get("/dashboard", s.handleDashboard)
		pr.Get("/athletes/{athleteID}/workouts", s.handleAthleteWorkouts)
		pr.Post("/athletes/{athleteID}/sync", s.handleTriggerSync)
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
	if err := s.Tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render template %s failed: %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	s.render(w, "login", map[string]any{"Title": "Login"})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	coachID := s.Sess.GetString(r.Context(), "coach_id")
	if coachID == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	cid := uuid.MustParse(coachID)

	athletes, err := s.Q.ListAthletesByCoach(r.Context(), cid)
	if err != nil {
		log.Printf("list athletes failed: %v", err)
		http.Error(w, "could not load athletes", 500)
		return
	}

	s.render(w, "dashboard", map[string]any{
		"Title":    "Dashboard",
		"Athletes": athletes,
	})
}

func (s *Server) handleCreateAthlete(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	name := strings.TrimSpace(r.Form.Get("name"))
	emailAddr := strings.TrimSpace(r.Form.Get("email"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}
	if emailAddr == "" {
		http.Error(w, "email required", 400)
		return
	}

	coachID := s.Sess.GetString(r.Context(), "coach_id")
	coachUUID := uuid.MustParse(coachID)

	a, err := s.Q.CreateAthlete(r.Context(), db.CreateAthleteParams{
		CoachID: coachUUID, // <- uuid.UUID, not pgtype.UUID
		Name:    name,
		Email:   pgtype.Text{String: emailAddr, Valid: true},
		Tz:      "Europe/Berlin",
	})
	if err != nil {
		log.Printf("create athlete failed: %v", err)
		http.Error(w, "could not create athlete", 500)
		return
	}

	invite := s.Invite.URL(coachID, a.ID.String(), 7*24*time.Hour)

	// Send invite email if sender is configured
	if s.Email != nil {
		inviteHTML := "<p>You have been invited to CoachGPT. Click the link below to connect to Strava:</p>" +
			"<p><a href=\"" + invite + "\">Connect to CoachGPT</a></p>"
		if err := s.Email.Send(emailAddr, "You're invited to CoachGPT", inviteHTML); err != nil {
			log.Printf("failed to send invite email to %s: %v", emailAddr, err)
		}
	}

	s.render(w, "invite_created", map[string]any{
		"Title":     "Invite Link",
		"InviteURL": invite,
		"Athlete":   a,
	})
}

func (s *Server) handleAthleteInvite(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	coachID, athleteID, err := s.Invite.Verify(tok)
	if err != nil {
		log.Printf("[invite] verify failed: %v", err)
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	s.render(w, "athlete_consent", map[string]any{
		"Title":     "Connect Strava",
		"CoachID":   coachID,
		"AthleteID": athleteID,
		"Token":     tok,
	})
}

func (s *Server) handleStravaStart(w http.ResponseWriter, r *http.Request) {
	aid := r.URL.Query().Get("aid")
	state := s.signState(aid, time.Now().Add(30*time.Minute))

	authURL := s.StravaConf.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("scope", "read,activity:read_all"),
		oauth2.SetAuthURLParam("approval_prompt", "auto"),
	)

	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) handleStravaCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	athleteID, ok := s.verifyState(state)
	if !ok {
		http.Error(w, "invalid state", 400)
		return
	}

	tok, err := s.StravaConf.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("strava token exchange failed: %v", err)
		http.Error(w, "could not exchange token", 500)
		return
	}

	id := uuid.MustParse(athleteID)
	expiry := tok.Expiry

	// Extract Strava athlete ID from token response
	var stravaAthleteID pgtype.Int8

	// Fetch athlete profile to get Strava athlete ID
	req, err := http.NewRequest("GET", "https://www.strava.com/api/v3/athlete", nil)
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close() //nolint:errcheck
			var athlete map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&athlete) == nil {
				if stravaID, ok := athlete["id"].(float64); ok {
					stravaAthleteID = pgtype.Int8{Int64: int64(stravaID), Valid: true}
				}
			}
		}
	}

	if err := s.Q.SetAthleteStravaTokens(r.Context(), db.SetAthleteStravaTokensParams{
		ID:                 id,
		StravaAthleteID:    stravaAthleteID,
		StravaAccessToken:  pgtype.Text{String: tok.AccessToken, Valid: true},
		StravaRefreshToken: pgtype.Text{String: tok.RefreshToken, Valid: true},
		StravaTokenExpiry:  pgtype.Timestamptz{Time: expiry, Valid: true},
	}); err != nil {
		log.Printf("set athlete strava token failed: %v", err)
		http.Error(w, "could not save token", 500)
		return
	}

	client := asynq.NewClient(asynq.RedisClientOpt{Addr: s.RedisAddr})
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Error closing asynq client: %v", closeErr)
		}
	}()
	payload, _ := json.Marshal(jobs.SyncStravaPayload{AthleteID: athleteID})
	task := asynq.NewTask(jobs.TaskSyncStrava, payload)

	// Configure retry policy for better reliability
	info, err := client.Enqueue(task,
		asynq.Queue("sync"),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
	)
	if err != nil {
		log.Printf("[asynq] enqueue failed: %v", err)
	} else {
		log.Printf("[asynq] enqueued task: id=%s queue=%s maxRetry=3", info.ID, info.Queue)
	}

	s.render(w, "athlete_connected", map[string]any{
		"Title": "Connected",
		"Msg":   "Strava connected! You can close this window.",
	})
}

func (s *Server) signState(athleteID string, exp time.Time) string {
	msg := athleteID + "|" + strconv.FormatInt(exp.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(s.StateSecret))
	mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	pl := base64.RawURLEncoding.EncodeToString([]byte(msg))
	return pl + "." + sig
}

func (s *Server) verifyState(state string) (athleteID string, ok bool) {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return
	}

	mac := hmac.New(sha256.New, []byte(s.StateSecret))
	mac.Write(payload)

	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return
	}

	fields := strings.SplitN(string(payload), "|", 2)
	if len(fields) != 2 {
		return
	}

	athleteID = fields[0]
	expUnix, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return
	}

	if time.Now().After(time.Unix(expUnix, 0)) {
		return
	}

	ok = true
	return
}

func (s *Server) handleMagicLink(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", 400)
		return
	}
	emailAddr := strings.TrimSpace(r.Form.Get("email"))
	if emailAddr == "" {
		http.Error(w, "email required", 400)
		return
	}

	url, err := s.issueMagicLink(r.Context(), emailAddr)
	if err != nil {
		log.Printf("[auth] issue link failed for %s: %v", emailAddr, err)
		http.Error(w, "could not issue link", 500)
		return
	}

	// Send email via configured sender
	if s.Email != nil {
		html := "<p>Click the link below to sign in:</p><p><a href=\"" + url + "\">Sign in</a></p>"
		if err := s.Email.Send(emailAddr, "Your CoachGPT sign-in link", html); err != nil {
			log.Printf("failed to send magic link email to %s: %v", emailAddr, err)
		}
	}

	log.Printf("[auth] magic link for %s: %s", emailAddr, url)
	s.render(w, "magic_sent", map[string]any{
		"Title": "Magic Link Sent", "Email": emailAddr, "URL": url,
	})
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	// Render public home/landing page
	s.render(w, "home", map[string]any{"Title": "CoachGPT — AI for coaches"})
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

func (s *Server) handleAthleteWorkouts(w http.ResponseWriter, r *http.Request) {
	coachID := r.Context().Value(appmw.CoachIDKey).(string)
	athleteID := chi.URLParam(r, "athleteID")

	// Parse athlete ID
	aid, err := uuid.Parse(athleteID)
	if err != nil {
		http.Error(w, "invalid athlete ID", http.StatusBadRequest)
		return
	}

	// Get athlete and verify it belongs to this coach
	athlete, err := s.Q.GetAthlete(r.Context(), aid)
	if err != nil {
		http.Error(w, "athlete not found", http.StatusNotFound)
		return
	}

	// Verify coach ownership
	if athlete.CoachID.String() != coachID {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	// Get workouts for this athlete (limit to 50 most recent)
	workouts, err := s.Q.ListWorkoutsByAthlete(r.Context(), db.ListWorkoutsByAthleteParams{
		AthleteID: aid,
		Limit:     50,
	})
	if err != nil {
		log.Printf("failed to list workouts for athlete %s: %v", athleteID, err)
		http.Error(w, "failed to load workouts", http.StatusInternalServerError)
		return
	}

	log.Printf("found workouts for athlete=%s: %d", athleteID, len(workouts))

	data := struct {
		Title    string
		Athlete  db.Athlete
		Workouts []db.ListWorkoutsByAthleteRow
	}{
		Title:    "Workouts - " + athlete.Name,
		Athlete:  athlete,
		Workouts: workouts,
	}

	s.render(w, "workouts", data)
}

func (s *Server) handleTriggerSync(w http.ResponseWriter, r *http.Request) {
	coachID := r.Context().Value(appmw.CoachIDKey).(string)
	athleteID := chi.URLParam(r, "athleteID")

	// Parse athlete ID
	aid, err := uuid.Parse(athleteID)
	if err != nil {
		http.Error(w, "invalid athlete ID", http.StatusBadRequest)
		return
	}

	// Get athlete and verify it belongs to this coach
	athlete, err := s.Q.GetAthlete(r.Context(), aid)
	if err != nil {
		http.Error(w, "athlete not found", http.StatusNotFound)
		return
	}

	// Verify coach ownership
	if athlete.CoachID.String() != coachID {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}

	// Verify athlete has Strava connection
	if !athlete.StravaAccessToken.Valid {
		http.Error(w, "athlete not connected to Strava", http.StatusBadRequest)
		return
	}

	// Queue sync job with force flag for manual syncs
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: s.RedisAddr})
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Error closing asynq client: %v", closeErr)
		}
	}()

	// For manual syncs, go back further to catch title/data changes
	forceFromTime := time.Now().AddDate(0, 0, -30).Unix() // 30 days back
	payload, err := json.Marshal(jobs.SyncStravaPayload{
		AthleteID: athleteID,
		SinceUnix: forceFromTime,
	})
	if err != nil {
		log.Printf("failed to marshal sync payload: %v", err)
		http.Error(w, "failed to queue sync job", http.StatusInternalServerError)
		return
	}

	task := asynq.NewTask(jobs.TaskSyncStrava, payload)
	info, err := client.Enqueue(task, asynq.Queue("sync"))
	if err != nil {
		log.Printf("failed to enqueue sync job: %v", err)
		http.Error(w, "failed to queue sync job", http.StatusInternalServerError)
		return
	}

	log.Printf("sync job queued for athlete %s: %s", athleteID, info.ID)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("sync job queued")); err != nil {
		log.Printf("Error writing sync response: %v", err)
	}
}

func (s *Server) handleInterestSubmit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	em := r.Form.Get("email")
	if em == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}

	// Send a notification email to the inbox you check (for now, send to the same address to capture in MailHog)
	if s.Email != nil {
		html := "<p>New interest sign-up: " + em + "</p>"
		_ = s.Email.Send(em, "CoachGPT interest signup", html)
	}

	s.render(w, "interest_submitted", map[string]any{"Title": "Thanks", "Email": em})
}
